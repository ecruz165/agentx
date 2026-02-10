package userdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetUserdataRoot_EnvOverride(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/test-userdata")
	root, err := GetUserdataRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != "/tmp/test-userdata" {
		t.Errorf("expected /tmp/test-userdata, got %s", root)
	}
}

func TestGetUserdataRoot_Default(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "")
	root, err := GetUserdataRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".agentx", "userdata")
	if root != expected {
		t.Errorf("expected %s, got %s", expected, root)
	}
}

func TestGetEnvDir(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")
	dir, err := GetEnvDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "/tmp/ud/env" {
		t.Errorf("expected /tmp/ud/env, got %s", dir)
	}
}

func TestGetProfilesDir(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")
	dir, err := GetProfilesDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "/tmp/ud/profiles" {
		t.Errorf("expected /tmp/ud/profiles, got %s", dir)
	}
}

func TestGetPreferencesPath(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")
	p, err := GetPreferencesPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "/tmp/ud/preferences.yaml" {
		t.Errorf("expected /tmp/ud/preferences.yaml, got %s", p)
	}
}

func TestGetSkillsDir(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")
	dir, err := GetSkillsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "/tmp/ud/skills" {
		t.Errorf("expected /tmp/ud/skills, got %s", dir)
	}
}

func TestGetVendorEnvPath(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")
	p, err := GetVendorEnvPath("aws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "/tmp/ud/env/aws.env" {
		t.Errorf("expected /tmp/ud/env/aws.env, got %s", p)
	}
}

func TestGetSkillRegistryPath(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")
	p, err := GetSkillRegistryPath("cloud/aws/ssm-lookup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "/tmp/ud/skills/cloud/aws/ssm-lookup" {
		t.Errorf("expected /tmp/ud/skills/cloud/aws/ssm-lookup, got %s", p)
	}
}

func TestPermissionConstants(t *testing.T) {
	if DirPermSecure != 0700 {
		t.Errorf("DirPermSecure: expected 0700, got %o", DirPermSecure)
	}
	if FilePermSecure != 0600 {
		t.Errorf("FilePermSecure: expected 0600, got %o", FilePermSecure)
	}
	if DirPermNormal != 0755 {
		t.Errorf("DirPermNormal: expected 0755, got %o", DirPermNormal)
	}
}
