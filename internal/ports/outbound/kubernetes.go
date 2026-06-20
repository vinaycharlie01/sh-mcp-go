package outbound

import (
	"context"

	"github.com/vinaycharlie01/sh-mcp-go/internal/domain/cluster"
)

// NamespaceSpec describes a namespace to create.
type NamespaceSpec struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

// CRDInfo describes a Custom Resource Definition.
type CRDInfo struct {
	Name    string
	Group   string
	Version string
	Kind    string
	YAML    string
}

// ResourceStatus summarises the status of a workload resource.
type ResourceStatus struct {
	Kind      string
	Name      string
	Namespace string
	Ready     bool
	Replicas  int32
	Available int32
	Message   string
}

// KubernetesPort is the outbound port for all Kubernetes API interactions.
// Implementations must use the Go SDK only — no kubectl, no exec.
type KubernetesPort interface {
	// EnsureNamespace creates a namespace if it doesn't exist.
	EnsureNamespace(ctx context.Context, spec NamespaceSpec) error

	// DeleteNamespace deletes a namespace and all its contents.
	DeleteNamespace(ctx context.Context, name string) error

	// ListNamespaces returns all namespaces in the cluster.
	ListNamespaces(ctx context.Context) ([]string, error)

	// ApplyCRD applies a CRD to the cluster using server-side apply.
	ApplyCRD(ctx context.Context, crd CRDInfo) error

	// ListCRDs returns all CRDs installed in the cluster.
	ListCRDs(ctx context.Context) ([]cluster.CRD, error)

	// CRDExists checks whether a CRD with the given name is installed.
	CRDExists(ctx context.Context, name string) (bool, error)

	// GetServerVersion returns the Kubernetes server version string.
	GetServerVersion(ctx context.Context) (string, error)

	// GetClusterInfo returns a full snapshot of cluster state.
	GetClusterInfo(ctx context.Context) (*cluster.ClusterInfo, error)

	// GetResourceHealth returns health status of resources in a namespace.
	GetResourceHealth(ctx context.Context, namespace, releaseName string) ([]cluster.ResourceHealth, error)

	// WaitForRollout waits until a deployment/statefulset is fully rolled out.
	WaitForRollout(ctx context.Context, kind, name, namespace string, timeoutSecs int) error

	// ValidateCluster runs prerequisite checks for deployment readiness.
	ValidateCluster(ctx context.Context) (*cluster.ValidationResult, error)

	// EstimateResources estimates required CPU/memory for a workload.
	EstimateResources(ctx context.Context, chartName, namespace string, replicas int) (*ResourceEstimate, error)
}

// ResourceEstimate holds estimated resource requirements.
type ResourceEstimate struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	StorageGB     float64
	Notes         []string
}
