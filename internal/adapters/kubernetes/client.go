package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/cluster"
	"github.com/vinaycharlie01/sh-mcp-go/internal/infrastructure/config"
	"github.com/vinaycharlie01/sh-mcp-go/internal/ports/outbound"
)

const (
	defaultStorageGB    = 5
	rolloutTickInterval = 5 * time.Second
)

// Client implements outbound.KubernetesPort using the official Go SDK.
// No kubectl is executed. All operations use client-go and controller-runtime APIs.
type Client struct {
	typed     kubernetes.Interface
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
	apiext    apiextclient.Interface
	restCfg   *rest.Config
	logger    *slog.Logger
	cfg       *config.KubernetesConfig
}

// NewClient builds a Kubernetes client from the provided configuration.
func NewClient(cfg *config.KubernetesConfig, logger *slog.Logger) (*Client, error) {
	restCfg, err := buildRESTConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building REST config: %w", err)
	}

	restCfg.QPS = cfg.QPS
	restCfg.Burst = cfg.Burst
	if cfg.Timeout > 0 {
		restCfg.Timeout = cfg.Timeout
	}

	typedClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating typed client: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	discClient, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}

	apiextClient, err := apiextclient.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating apiextensions client: %w", err)
	}

	return &Client{
		typed:     typedClient,
		dynamic:   dynClient,
		discovery: discClient,
		apiext:    apiextClient,
		restCfg:   restCfg,
		logger:    logger,
		cfg:       cfg,
	}, nil
}

// EnsureNamespace creates a namespace if it does not already exist.
func (c *Client) EnsureNamespace(ctx context.Context, spec outbound.NamespaceSpec) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
	}

	_, err := c.typed.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		c.logger.Debug("namespace already exists", slog.String("namespace", spec.Name))
		return nil
	}
	if err != nil {
		return fmt.Errorf("creating namespace %q: %w", spec.Name, err)
	}
	c.logger.Info("namespace created", slog.String("namespace", spec.Name))
	return nil
}

// DeleteNamespace removes a namespace and all its contents.
func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.typed.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// ListNamespaces returns all namespaces.
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	list, err := c.typed.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}
	names := make([]string, len(list.Items))
	for i, ns := range list.Items {
		names[i] = ns.Name
	}
	return names, nil
}

// ApplyCRD applies a CRD using server-side apply.
func (c *Client) ApplyCRD(ctx context.Context, crdInfo outbound.CRDInfo) error {
	crd := &apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdInfo.Name,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: crdInfo.Group,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind: crdInfo.Kind,
			},
			Scope: apiextv1.NamespaceScoped,
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:    crdInfo.Version,
					Served:  true,
					Storage: true,
					Schema: &apiextv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextv1.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	data, err := marshalCRD(crd)
	if err != nil {
		return fmt.Errorf("marshaling CRD: %w", err)
	}

	_, err = c.dynamic.Resource(gvr).Patch(
		ctx,
		crdInfo.Name,
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: "sh-mcp-go", Force: boolPtr(true)},
	)
	return err
}

// ListCRDs returns all installed CRDs.
func (c *Client) ListCRDs(ctx context.Context) ([]cluster.CRD, error) {
	list, err := c.apiext.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing CRDs: %w", err)
	}

	crds := make([]cluster.CRD, 0, len(list.Items))
	for _, item := range list.Items {
		ver := ""
		if len(item.Spec.Versions) > 0 {
			ver = item.Spec.Versions[0].Name
		}
		crds = append(crds, cluster.CRD{
			Name:    item.Name,
			Group:   item.Spec.Group,
			Version: ver,
			Kind:    item.Spec.Names.Kind,
		})
	}
	return crds, nil
}

// CRDExists checks if a CRD is installed.
func (c *Client) CRDExists(ctx context.Context, name string) (bool, error) {
	_, err := c.apiext.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetServerVersion returns the Kubernetes server version.
func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	ver, err := c.discovery.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("getting server version: %w", err)
	}
	return ver.GitVersion, nil
}

// GetClusterInfo returns a comprehensive cluster state snapshot.
func (c *Client) GetClusterInfo(ctx context.Context) (*cluster.ClusterInfo, error) {
	ver, err := c.GetServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	namespaces, err := c.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	nodes, err := c.listNodes(ctx)
	if err != nil {
		return nil, err
	}

	crds, err := c.ListCRDs(ctx)
	if err != nil {
		crds = nil
	}

	return &cluster.ClusterInfo{
		ServerVersion: ver,
		Nodes:         nodes,
		Namespaces:    namespaces,
		CRDs:          crds,
		CollectedAt:   time.Now().UTC(),
	}, nil
}

// GetResourceHealth returns health of workloads related to a Helm release.
func (c *Client) GetResourceHealth(ctx context.Context, namespace, releaseName string) ([]cluster.ResourceHealth, error) {
	var health []cluster.ResourceHealth

	// Check deployments
	deployments, err := c.typed.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err == nil {
		for _, d := range deployments.Items {
			ready := d.Status.ReadyReplicas == *d.Spec.Replicas
			status := cluster.HealthStatusHealthy
			if !ready {
				status = cluster.HealthStatusDegraded
			}
			health = append(health, cluster.ResourceHealth{
				Kind:      "Deployment",
				Name:      d.Name,
				Namespace: d.Namespace,
				Status:    status,
				Ready:     ready,
				Age:       time.Since(d.CreationTimestamp.Time),
			})
		}
	}

	// Check statefulsets
	statefulsets, err := c.typed.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err == nil {
		for _, ss := range statefulsets.Items {
			ready := ss.Status.ReadyReplicas == *ss.Spec.Replicas
			status := cluster.HealthStatusHealthy
			if !ready {
				status = cluster.HealthStatusDegraded
			}
			health = append(health, cluster.ResourceHealth{
				Kind:      "StatefulSet",
				Name:      ss.Name,
				Namespace: ss.Namespace,
				Status:    status,
				Ready:     ready,
				Age:       time.Since(ss.CreationTimestamp.Time),
			})
		}
	}

	return health, nil
}

// WaitForRollout waits until a workload is fully rolled out.
func (c *Client) WaitForRollout(ctx context.Context, kind, name, namespace string, timeoutSecs int) error {
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	ticker := time.NewTicker(rolloutTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for %s/%s rollout after %ds", kind, name, timeoutSecs)
			}
			ready, err := c.isWorkloadReady(ctx, kind, name, namespace)
			if err != nil {
				c.logger.Warn("checking workload ready", slog.String("error", err.Error()))

				continue
			}
			if ready {
				return nil
			}
		}
	}
}

// ValidateCluster checks cluster prerequisites.
func (c *Client) ValidateCluster(ctx context.Context) (*cluster.ValidationResult, error) {
	result := &cluster.ValidationResult{
		Valid:     true,
		CheckedAt: time.Now().UTC(),
	}

	ver, err := c.GetServerVersion(ctx)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("cannot reach API server: %v", err))
		return result, nil
	}
	c.logger.Info("cluster reachable", slog.String("version", ver))

	nodes, err := c.listNodes(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cannot list nodes: %v", err))
		result.Valid = false
	} else if len(nodes) == 0 {
		result.Warnings = append(result.Warnings, "no nodes found in cluster")
	}

	notReady := 0
	for _, n := range nodes {
		if n.Status != cluster.NodeStatusReady {
			notReady++
		}
	}
	if notReady > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d node(s) not ready", notReady))
	}

	return result, nil
}

// EstimateResources returns resource estimates for a workload.
func (c *Client) EstimateResources(_ context.Context, chartName, _ string, replicas int) (*outbound.ResourceEstimate, error) {
	estimates := map[string]*outbound.ResourceEstimate{
		"prometheus": {
			CPURequest:    fmt.Sprintf("%dm", 100*replicas),
			CPULimit:      fmt.Sprintf("%dm", 500*replicas),
			MemoryRequest: fmt.Sprintf("%dMi", 256*replicas),
			MemoryLimit:   fmt.Sprintf("%dMi", 1024*replicas),
			StorageGB:     float64(10 * replicas),
		},
		"grafana": {
			CPURequest: "50m", CPULimit: "200m",
			MemoryRequest: "128Mi", MemoryLimit: "512Mi",
			StorageGB: 1,
		},
		"redis": {
			CPURequest:    fmt.Sprintf("%dm", 50*replicas),
			CPULimit:      fmt.Sprintf("%dm", 200*replicas),
			MemoryRequest: fmt.Sprintf("%dMi", 128*replicas),
			MemoryLimit:   fmt.Sprintf("%dMi", 512*replicas),
			StorageGB:     float64(5 * replicas),
		},
	}

	if est, ok := estimates[chartName]; ok {
		return est, nil
	}
	return &outbound.ResourceEstimate{
		CPURequest:    "100m",
		CPULimit:      "500m",
		MemoryRequest: "128Mi",
		MemoryLimit:   "512Mi",
		StorageGB:     defaultStorageGB,
		Notes:         []string{"estimate based on defaults; tune for production"},
	}, nil
}

// listNodes returns node status from the cluster.
func (c *Client) listNodes(ctx context.Context) ([]cluster.Node, error) {
	nodeList, err := c.typed.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodes := make([]cluster.Node, 0, len(nodeList.Items))
	for _, n := range nodeList.Items {
		status := cluster.NodeStatusUnknown
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					status = cluster.NodeStatusReady
				} else {
					status = cluster.NodeStatusNotReady
				}
			}
		}

		var roles []string
		for label := range n.Labels {
			if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
				roles = append(roles, "control-plane")
			}
			if label == "node-role.kubernetes.io/worker" {
				roles = append(roles, "worker")
			}
		}
		if len(roles) == 0 {
			roles = []string{"worker"}
		}

		nodes = append(nodes, cluster.Node{
			Name:   n.Name,
			Status: status,
			Roles:  roles,
			Labels: n.Labels,
			Capacity: cluster.ResourceCapacity{
				CPU:    n.Status.Capacity.Cpu().String(),
				Memory: n.Status.Capacity.Memory().String(),
				Pods:   n.Status.Capacity.Pods().Value(),
			},
			Allocatable: cluster.ResourceCapacity{
				CPU:    n.Status.Allocatable.Cpu().String(),
				Memory: n.Status.Allocatable.Memory().String(),
				Pods:   n.Status.Allocatable.Pods().Value(),
			},
		})
	}
	return nodes, nil
}

// isWorkloadReady checks if a named workload is ready.
func (c *Client) isWorkloadReady(ctx context.Context, kind, name, namespace string) (bool, error) {
	switch kind {
	case "Deployment":
		d, err := c.typed.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return d.Status.ReadyReplicas == *d.Spec.Replicas, nil
	case "StatefulSet":
		ss, err := c.typed.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return ss.Status.ReadyReplicas == *ss.Spec.Replicas, nil
	case "DaemonSet":
		ds, err := c.typed.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return ds.Status.NumberReady == ds.Status.DesiredNumberScheduled, nil
	}
	return false, fmt.Errorf("unsupported kind %q", kind)
}

// buildRESTConfig builds a *rest.Config from our application config.
func buildRESTConfig(cfg *config.KubernetesConfig) (*rest.Config, error) {
	if cfg.InCluster {
		return rest.InClusterConfig()
	}
	if cfg.KubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
}

func boolPtr(b bool) *bool { return &b }
