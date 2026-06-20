package deployment

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

var releaseNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,52}[a-z0-9]$|^[a-z0-9]$`)
var namespaceRE = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,61}[a-z0-9]$|^[a-z0-9]$`)

// ID uniquely identifies a Deployment aggregate.
type ID string

func (id ID) String() string { return string(id) }

// ReleaseName is the Helm release name, validated to k8s naming rules.
type ReleaseName string

func (r ReleaseName) String() string { return string(r) }

func (r ReleaseName) Validate() error {
	if string(r) == "" {
		return fmt.Errorf("release name must not be empty")
	}
	if !releaseNameRE.MatchString(string(r)) {
		return fmt.Errorf("release name %q is invalid: must be lowercase alphanumeric and hyphens, 1-53 chars", r)
	}

	return nil
}

// Namespace is a validated Kubernetes namespace name.
type Namespace string

func (n Namespace) String() string { return string(n) }

func (n Namespace) Validate() error {
	if string(n) == "" {
		return fmt.Errorf("namespace must not be empty")
	}
	if !namespaceRE.MatchString(string(n)) {
		return fmt.Errorf("namespace %q is invalid", n)
	}

	return nil
}

// ChartSource indicates where a chart originates.
type ChartSource string

const (
	ChartSourceRepo ChartSource = "REPO"
	ChartSourceOCI  ChartSource = "OCI"
	ChartSourcePath ChartSource = "PATH"
)

// ChartReference identifies a Helm chart with its repo, name, and version.
type ChartReference struct {
	Name    string
	RepoURL string
	Version string
	Source  ChartSource
}

func (c ChartReference) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("chart name must not be empty")
	}
	if c.Source == ChartSourceRepo && strings.TrimSpace(c.RepoURL) == "" {
		return fmt.Errorf("repo URL must not be empty for REPO source")
	}
	if c.Version != "" && c.Version != "latest" {
		if _, err := semver.NewVersion(c.Version); err != nil {
			return fmt.Errorf("chart version %q is not a valid semver: %w", c.Version, err)
		}
	}
	return nil
}

func (c ChartReference) String() string {
	if c.Version != "" {
		return fmt.Sprintf("%s:%s", c.Name, c.Version)
	}
	return c.Name
}

// Values holds user-supplied Helm values overrides.
type Values map[string]any

// Merge produces a new Values by merging overrides on top of v.
func (v Values) Merge(overrides Values) Values {
	result := make(Values, len(v)+len(overrides))
	for k, val := range v {
		result[k] = val
	}
	for k, val := range overrides {
		result[k] = val
	}
	return result
}

// HistoryEntry records a past deployment state.
type HistoryEntry struct {
	Version     int
	Status      Status
	ChartRef    ChartReference
	HelmVersion int
	Timestamp   time.Time
	Notes       string
}
