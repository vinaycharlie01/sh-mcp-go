package cluster

import "time"

// NodeStatus represents the health of a cluster node.
type NodeStatus string

const (
	NodeStatusReady    NodeStatus = "READY"
	NodeStatusNotReady NodeStatus = "NOT_READY"
	NodeStatusUnknown  NodeStatus = "UNKNOWN"
)

// Node represents a Kubernetes cluster node.
type Node struct {
	Name       string
	Status     NodeStatus
	Roles      []string
	Taints     []string
	Labels     map[string]string
	Capacity   ResourceCapacity
	Allocatable ResourceCapacity
}

// ResourceCapacity holds CPU and memory capacity values.
type ResourceCapacity struct {
	CPU    string
	Memory string
	Pods   int64
}

// Release summarises a Helm release installed in the cluster.
type Release struct {
	Name       string
	Namespace  string
	Chart      string
	Version    string
	AppVersion string
	Status     string
	UpdatedAt  time.Time
}

// CRD summarises a Custom Resource Definition present in the cluster.
type CRD struct {
	Name    string
	Group   string
	Version string
	Kind    string
}

// ClusterInfo aggregates the observable state of a Kubernetes cluster.
type ClusterInfo struct {
	ServerVersion string
	Nodes         []Node
	Namespaces    []string
	Releases      []Release
	CRDs          []CRD
	CollectedAt   time.Time
}

// HealthStatus represents the overall health of a deployed resource.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "HEALTHY"
	HealthStatusDegraded  HealthStatus = "DEGRADED"
	HealthStatusUnhealthy HealthStatus = "UNHEALTHY"
	HealthStatusUnknown   HealthStatus = "UNKNOWN"
)

// ResourceHealth represents the health of a single Kubernetes resource.
type ResourceHealth struct {
	Kind      string
	Name      string
	Namespace string
	Status    HealthStatus
	Message   string
	Ready     bool
	Age       time.Duration
}

// ValidationResult holds the outcome of a cluster validation check.
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
	CheckedAt time.Time
}
