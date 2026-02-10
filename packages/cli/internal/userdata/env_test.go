package userdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRedactValue_SensitiveKeys(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"GITHUB_TOKEN", "ghp_abcdef123456", "ghp_***"},
		{"AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI", "wJal***"},
		{"DB_PASSWORD", "hunter2", "hunt***"},
		{"API_KEY", "sk-12345", "sk-1***"},
		{"SPLUNK_CREDENTIAL", "abc", "***"},
		{"github_token", "ghp_abcdef", "ghp_***"}, // case insensitive via key
		{"LOG_LEVEL", "info", "info"},               // not sensitive
		{"OUTPUT_FORMAT", "json", "json"},           // not sensitive
		{"AWS_REGION", "us-east-1", "us-east-1"},   // not sensitive
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := RedactValue(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("RedactValue(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

func TestRedactValue_ShortValues(t *testing.T) {
	result := RedactValue("MY_SECRET", "ab")
	if result != "***" {
		t.Errorf("expected ***, got %s", result)
	}

	result = RedactValue("MY_SECRET", "")
	if result != "***" {
		t.Errorf("expected ***, got %s", result)
	}
}

func TestParseEnvFile(t *testing.T) {
	tmp := t.TempDir()
	envFile := filepath.Join(tmp, "test.env")

	content := `# This is a comment
LOG_LEVEL=info
OUTPUT_FORMAT=json

# Another comment
CONNECTION_STRING=host=localhost port=5432 dbname=mydb
EMPTY_VALUE=
`
	os.WriteFile(envFile, []byte(content), 0644)

	entries, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile failed: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	if entries[0].Key != "LOG_LEVEL" || entries[0].Value != "info" {
		t.Errorf("entry 0: got %s=%s", entries[0].Key, entries[0].Value)
	}

	// Value with = sign inside.
	if entries[2].Key != "CONNECTION_STRING" || entries[2].Value != "host=localhost port=5432 dbname=mydb" {
		t.Errorf("entry 2: got %s=%s", entries[2].Key, entries[2].Value)
	}

	// Empty value.
	if entries[3].Key != "EMPTY_VALUE" || entries[3].Value != "" {
		t.Errorf("entry 3: got %s=%s", entries[3].Key, entries[3].Value)
	}
}

func TestResolveEnvTarget_Vendor(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")

	path, err := ResolveEnvTarget("aws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/tmp/ud/env/aws.env" {
		t.Errorf("expected /tmp/ud/env/aws.env, got %s", path)
	}
}

func TestResolveEnvTarget_SkillPath(t *testing.T) {
	t.Setenv("AGENTX_USERDATA", "/tmp/ud")

	path, err := ResolveEnvTarget("cloud/aws/ssm-lookup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/tmp/ud/skills/cloud/aws/ssm-lookup/tokens.env" {
		t.Errorf("expected /tmp/ud/skills/cloud/aws/ssm-lookup/tokens.env, got %s", path)
	}
}

func TestListEnvFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	// Create shared env files.
	envDir := filepath.Join(tmp, "env")
	os.MkdirAll(envDir, 0700)
	os.WriteFile(filepath.Join(envDir, "default.env"), []byte("X=1"), 0600)
	os.WriteFile(filepath.Join(envDir, "aws.env"), []byte("Y=2"), 0600)

	// Create skill-specific tokens.env.
	skillDir := filepath.Join(tmp, "skills", "cloud", "aws", "ssm-lookup")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "tokens.env"), []byte("Z=3"), 0600)

	shared, skillSpecific, err := ListEnvFiles()
	if err != nil {
		t.Fatalf("ListEnvFiles failed: %v", err)
	}

	if len(shared) != 2 {
		t.Errorf("expected 2 shared files, got %d", len(shared))
	}
	if len(skillSpecific) != 1 {
		t.Errorf("expected 1 skill-specific file, got %d", len(skillSpecific))
	}
}
