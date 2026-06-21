//go:build mage

// Mage build file for sh-mcp-go.
// Powered by nava (https://github.com/nirantaraai/nava).
//
// Usage:
//
//	go install github.com/magefile/mage@latest
//	mage -l          # list targets
//	mage build       # compile for current platform
//	mage test        # run tests
//	mage lint        # run golangci-lint
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/magefile/mage/mg"
	dockermagex "github.com/nirantaraai/nava/mage/docker"
	gitmagex "github.com/nirantaraai/nava/mage/git"
	gomagex "github.com/nirantaraai/nava/mage/golang"
)

const versionPkg = "github.com/vinaycharlie01/sh-mcp-go/pkg/version"

// init loads all YAML configs once before any target runs.
func init() {
	_ = gomagex.LoadConfig("go.yaml")
	_ = dockermagex.LoadConfig("docker.yaml")
}

// ---- Go targets --------------------------------------------------------

// Build compiles sh-mcp-go for the current platform with git version ldflags.
func Build() error {
	version, _ := gitmagex.GetVersion()
	commit, _ := gitmagex.GetShortCommitSHA()
	date := time.Now().UTC().Format(time.RFC3339)

	ldf := fmt.Sprintf("-s -w -X %s.Version=%s -X %s.Commit=%s -X %s.BuildDate=%s",
		versionPkg, version, versionPkg, commit, versionPkg, date,
	)

	if err := os.MkdirAll("dist", 0o755); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-ldflags", ldf, "-o", "dist/sh-mcp-go", "./cmd/sh-mcp-go")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Test runs the unit test suite (config: go.yaml → test).
func Test() error { return gomagex.Test() }

// Lint runs golangci-lint (config: go.yaml → lint).
func Lint() error { return gomagex.Lint() }

// Vet runs go vet (config: go.yaml → vet).
func Vet() error { return gomagex.Vet() }

// Setup downloads Go modules (config: go.yaml → setup).
func Setup() error { return gomagex.Setup() }

// Race runs tests with race detection.
func Race() error {
	cmd := exec.Command("go", "test", "-race", "./...", "-short", "-timeout=120s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Coverage runs tests with coverage profiling and writes coverage.out.
func Coverage() error {
	cmd := exec.Command("go", "test", "./...",
		"-coverprofile=coverage.out", "-covermode=atomic", "-timeout=120s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Bench runs benchmarks.
func Bench() error {
	cmd := exec.Command("go", "test", "./...",
		"-bench=.", "-benchmem", "-run=^$", "-timeout=120s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Govulncheck runs govulncheck for vulnerability scanning.
func Govulncheck() error {
	cmd := exec.Command("govulncheck", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// BuildLinux cross-compiles for linux/amd64 and linux/arm64 (used by Docker multi-platform builds).
func BuildLinux() error {
	version, _ := gitmagex.GetVersion()
	commit, _ := gitmagex.GetShortCommitSHA()
	date := time.Now().UTC().Format(time.RFC3339)

	ldf := fmt.Sprintf("-s -w -X %s.Version=%s -X %s.Commit=%s -X %s.BuildDate=%s",
		versionPkg, version, versionPkg, commit, versionPkg, date,
	)

	for _, arch := range []string{"amd64", "arm64"} {
		outDir := filepath.Join("dist", "linux_"+arch)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}

		out := filepath.Join(outDir, "sh-mcp-go")
		fmt.Printf("building linux/%s → %s\n", arch, out)

		cmd := exec.Command("go", "build", "-ldflags", ldf, "-o", out, "./cmd/sh-mcp-go")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+arch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build linux/%s: %w", arch, err)
		}
	}
	return nil
}

// Clean removes build artefacts.
func Clean() error {
	fmt.Println("cleaning dist/")
	return os.RemoveAll("dist")
}

// ---- Docker targets ----------------------------------------------------

// Docker namespace for container operations.
type Docker mg.Namespace

// Build builds a multi-platform container image (config: docker.yaml → buildxBuild).
func (Docker) Build() error { return dockermagex.BuildxBuild() }

// Push pushes the image to the registry (config: docker.yaml → push).
func (Docker) Push() error { return dockermagex.Push() }

// Login logs in to the container registry (config: docker.yaml → login).
func (Docker) Login() error { return dockermagex.Login() }

// ---- Release target ----------------------------------------------------

// Release creates a GitHub release via goreleaser.
func Release() error {
	cmd := exec.Command("goreleaser", "release", "--clean", "--config", ".goreleaser.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
