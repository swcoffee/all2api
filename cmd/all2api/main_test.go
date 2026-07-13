package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseConfigFlagDefaultAndOverrides(t *testing.T) {
	cfg, provided, rest, err := parseConfigFlag([]string{"login", "zed"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg != defaultConfigPath {
		t.Fatalf("cfg = %q", cfg)
	}
	if provided {
		t.Fatal("expected config not to be marked provided")
	}
	if len(rest) != 2 || rest[0] != "login" || rest[1] != "zed" {
		t.Fatalf("rest = %#v", rest)
	}

	t.Setenv("ALL2API_CONFIG", "from-env.yaml")
	cfg, provided, _, err = parseConfigFlag([]string{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg != "from-env.yaml" {
		t.Fatalf("cfg = %q", cfg)
	}
	if provided {
		t.Fatal("expected env config not to be marked provided")
	}
}

func TestParseConfigFlagRecognizesConfigFlag(t *testing.T) {
	cfg, provided, rest, err := parseConfigFlag([]string{"-config", "a.yaml", "login", "tabbit"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg != "a.yaml" {
		t.Fatalf("cfg = %q", cfg)
	}
	if !provided {
		t.Fatal("expected config to be marked provided")
	}
	if len(rest) != 2 || rest[0] != "login" || rest[1] != "tabbit" {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestResolveConfigPathPrefersDockerMountedConfig(t *testing.T) {
	wd := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Create ./config/config.yaml
	if err := os.MkdirAll(filepath.Join(wd, "config"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wd, "config", "config.yaml"), []byte("server: {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := resolveConfigPath("config.yaml", false)
	if got != filepath.Join("config", "config.yaml") {
		t.Fatalf("resolved = %q", got)
	}

	// Explicit flag should not be overridden.
	got = resolveConfigPath("explicit.yaml", true)
	if got != "explicit.yaml" {
		t.Fatalf("resolved = %q", got)
	}
}

func TestUpdateConfigDoesNotPanicWithQuotedToken(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("server: {}\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	token := `abc"def`

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("updateConfig panicked: %v", r)
		}
	}()

	if err := updateConfig(cfgPath, "swcoffee", token); err != nil {
		t.Fatalf("updateConfig: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	body := root.Content[0]

	var upstreams *yaml.Node
	for i := 0; i < len(body.Content); i += 2 {
		if body.Content[i].Value == "upstreams" {
			upstreams = body.Content[i+1]
			break
		}
	}
	if upstreams == nil {
		t.Fatal("missing upstreams")
	}

	var target *yaml.Node
	for i := 0; i < len(upstreams.Content); i += 2 {
		if upstreams.Content[i].Value == "swcoffee" {
			target = upstreams.Content[i+1]
			break
		}
	}
	if target == nil {
		t.Fatal("missing upstream swcoffee")
	}

	var auth *yaml.Node
	for i := 0; i < len(target.Content); i += 2 {
		if target.Content[i].Value == "auth" {
			auth = target.Content[i+1]
			break
		}
	}
	if auth == nil {
		t.Fatal("missing auth")
	}

	var gotToken string
	for i := 0; i < len(auth.Content); i += 2 {
		if auth.Content[i].Value == "token" {
			gotToken = auth.Content[i+1].Value
			break
		}
	}
	if gotToken != token {
		t.Fatalf("token = %q, want %q", gotToken, token)
	}
}
