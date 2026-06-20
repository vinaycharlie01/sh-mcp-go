//go:build mage

// Mage build file — driven by build.yaml (nava-inspired pattern).
// Install mage: go install github.com/magefile/mage@latest
// Usage: mage <target>   e.g.  mage build   mage buildAll   mage test
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ---- config types --------------------------------------------------------

type buildYAML struct {
	Binary  binaryConfig  `yaml:"binary"`
	Docker  dockerConfig  `yaml:"docker"`
	Release releaseConfig `yaml:"release"`
}

type binaryConfig struct {
	Name       string     `yaml:"name"`
	Main       string     `yaml:"main"`
	OutputDir  string     `yaml:"output_dir"`
	CGOEnabled bool       `yaml:"cgo_enabled"`
	VersionPkg string     `yaml:"version_pkg"`
	Platforms  []platform `yaml:"platforms"`
}

type platform struct {
	OS   string `yaml:"os"`
	Arch string `yaml:"arch"`
}

type dockerConfig struct {
	Image      string   `yaml:"image"`
	Dockerfile string   `yaml:"dockerfile"`
	Platforms  []string `yaml:"platforms"`
}

type releaseConfig struct {
	Tool   string `yaml:"tool"`
	Config string `yaml:"config"`
}

// ---- helpers -------------------------------------------------------------

func loadConfig() (buildYAML, error) {
	var cfg buildYAML

	data, err := os.ReadFile("build.yaml")
	if err != nil {
		return cfg, fmt.Errorf("reading build.yaml: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing build.yaml: %w", err)
	}

	return cfg, nil
}

func gitStr(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(out))
}

func ldflags(pkg string) string {
	version := gitStr("describe", "--tags", "--always", "--dirty")
	commit := gitStr("rev-parse", "--short", "HEAD")
	date := time.Now().UTC().Format(time.RFC3339)

	return fmt.Sprintf(
		"-s -w -X %s.Version=%s -X %s.Commit=%s -X %s.BuildDate=%s",
		pkg, version, pkg, commit, pkg, date,
	)
}

func run(env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ---- targets -------------------------------------------------------------

// Build compiles sh-mcp-go for the current platform.
func Build() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.Binary.OutputDir, 0o755); err != nil {
		return err
	}

	out := filepath.Join(cfg.Binary.OutputDir, cfg.Binary.Name)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}

	cgo := "0"
	if cfg.Binary.CGOEnabled {
		cgo = "1"
	}

	fmt.Printf("build  %s  →  %s\n", cfg.Binary.Main, out)

	return run(
		[]string{"CGO_ENABLED=" + cgo},
		"go", "build", "-ldflags", ldflags(cfg.Binary.VersionPkg),
		"-o", out, cfg.Binary.Main,
	)
}

// BuildAll cross-compiles for every platform listed in build.yaml.
func BuildAll() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ldf := ldflags(cfg.Binary.VersionPkg)
	cgo := "0"
	if cfg.Binary.CGOEnabled {
		cgo = "1"
	}

	for _, p := range cfg.Binary.Platforms {
		name := cfg.Binary.Name
		if p.OS == "windows" {
			name += ".exe"
		}

		out := filepath.Join(cfg.Binary.OutputDir,
			fmt.Sprintf("%s_%s_%s", cfg.Binary.Name, p.OS, p.Arch),
			name,
		)

		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}

		fmt.Printf("build  %s/%s  →  %s\n", p.OS, p.Arch, out)

		if err := run(
			[]string{
				"CGO_ENABLED=" + cgo,
				"GOOS=" + p.OS,
				"GOARCH=" + p.Arch,
			},
			"go", "build", "-ldflags", ldf, "-o", out, cfg.Binary.Main,
		); err != nil {
			return fmt.Errorf("build %s/%s: %w", p.OS, p.Arch, err)
		}
	}

	return nil
}

// Test runs the unit test suite.
func Test() error {
	return run(nil, "go", "test", "./...", "-v", "-short", "-timeout", "120s")
}

// Lint runs golangci-lint.
func Lint() error {
	return run(nil, "golangci-lint", "run", "--timeout=10m")
}

// Docker builds the container image for all configured platforms.
func Docker() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	platforms := strings.Join(cfg.Docker.Platforms, ",")
	fmt.Printf("docker build  image=%s  platforms=%s\n", cfg.Docker.Image, platforms)

	return run(nil,
		"docker", "buildx", "build",
		"--platform", platforms,
		"-t", cfg.Docker.Image+":latest",
		"-f", cfg.Docker.Dockerfile,
		".",
	)
}

// Release creates a GitHub release via goreleaser.
func Release() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("release  tool=%s  config=%s\n", cfg.Release.Tool, cfg.Release.Config)

	return run(nil, cfg.Release.Tool, "release", "--clean", "--config", cfg.Release.Config)
}

// Clean removes all build artifacts.
func Clean() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("clean  %s/\n", cfg.Binary.OutputDir)

	return os.RemoveAll(cfg.Binary.OutputDir)
}
