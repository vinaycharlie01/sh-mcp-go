package outbound

import "context"

// OperatorStatus represents the lifecycle state of a Kubernetes operator.
type OperatorStatus string

const (
	OperatorStatusInstalled   OperatorStatus = "INSTALLED"
	OperatorStatusUpgrading   OperatorStatus = "UPGRADING"
	OperatorStatusFailed      OperatorStatus = "FAILED"
	OperatorStatusUninstalled OperatorStatus = "UNINSTALLED"
	OperatorStatusUnknown     OperatorStatus = "UNKNOWN"
)

// OperatorInfo describes an installed operator.
type OperatorInfo struct {
	Name      string
	Namespace string
	Version   string
	Channel   string
	Source    string
	Status    OperatorStatus
}

// OperatorInstallRequest carries parameters for installing an operator.
type OperatorInstallRequest struct {
	Name             string
	Namespace        string
	Channel          string
	Source           string
	SourceNamespace  string
	InstallPlanApproval string // "Automatic" or "Manual"
	StartingCSV      string
	TargetNamespaces []string
}

// OperatorUpgradeRequest carries parameters for upgrading an operator.
type OperatorUpgradeRequest struct {
	Name      string
	Namespace string
	Channel   string
	Version   string
}

// OperatorPort is the outbound port for Kubernetes operator lifecycle management.
type OperatorPort interface {
	// InstallOperator installs a Kubernetes operator via OLM or CRD-based install.
	InstallOperator(ctx context.Context, req OperatorInstallRequest) (*OperatorInfo, error)

	// UpgradeOperator upgrades an installed operator to a new channel/version.
	UpgradeOperator(ctx context.Context, req OperatorUpgradeRequest) (*OperatorInfo, error)

	// RollbackOperator reverts an operator upgrade.
	RollbackOperator(ctx context.Context, name, namespace string) error

	// DeleteOperator removes an operator from the cluster.
	DeleteOperator(ctx context.Context, name, namespace string) error

	// GetOperator returns information about an installed operator.
	GetOperator(ctx context.Context, name, namespace string) (*OperatorInfo, error)

	// ListOperators lists all installed operators.
	ListOperators(ctx context.Context, namespace string) ([]*OperatorInfo, error)

	// ValidateOperatorPrerequisites checks that OLM and required CRDs are present.
	ValidateOperatorPrerequisites(ctx context.Context) error

	// WaitForOperatorReady waits until the operator pod is ready.
	WaitForOperatorReady(ctx context.Context, name, namespace string, timeoutSecs int) error
}
