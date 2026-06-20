package deployment

import "fmt"

// ErrInvalidTransition is returned when a state machine transition is invalid.
type ErrInvalidTransition struct {
	From Status
	To   Status
}

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid deployment state transition: %s -> %s", e.From, e.To)
}

// ErrDeploymentNotFound is returned when a deployment cannot be found by its ID.
type ErrDeploymentNotFound struct {
	ID ID
}

func (e ErrDeploymentNotFound) Error() string {
	return fmt.Sprintf("deployment %q not found", e.ID)
}

// ErrReleaseNotFound is returned when a Helm release cannot be found.
type ErrReleaseNotFound struct {
	Name      string
	Namespace string
}

func (e ErrReleaseNotFound) Error() string {
	return fmt.Sprintf("helm release %q in namespace %q not found", e.Name, e.Namespace)
}
