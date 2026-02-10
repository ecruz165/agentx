package integrations

import "testing"

func TestParseToolName_OpenCode(t *testing.T) {
	name, ok := ParseToolName("opencode")
	if !ok {
		t.Fatal("ParseToolName(\"opencode\") returned false, want true")
	}
	if name != OpenCode {
		t.Fatalf("ParseToolName(\"opencode\") = %q, want %q", name, OpenCode)
	}
}

func TestParseToolName_AllKnown(t *testing.T) {
	cases := []struct {
		input string
		want  ToolName
	}{
		{"claude-code", ClaudeCode},
		{"copilot", Copilot},
		{"augment", Augment},
		{"opencode", OpenCode},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			name, ok := ParseToolName(tc.input)
			if !ok {
				t.Fatalf("ParseToolName(%q) returned false, want true", tc.input)
			}
			if name != tc.want {
				t.Fatalf("ParseToolName(%q) = %q, want %q", tc.input, name, tc.want)
			}
		})
	}
}

func TestParseToolName_Invalid(t *testing.T) {
	cases := []string{"unknown", "", "OPENCODE", "open-code", "claude"}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			name, ok := ParseToolName(input)
			if ok {
				t.Fatalf("ParseToolName(%q) returned true, want false", input)
			}
			if name != "" {
				t.Fatalf("ParseToolName(%q) = %q, want empty string", input, name)
			}
		})
	}
}

func TestAllTools_ContainsOpenCode(t *testing.T) {
	tools := AllTools()
	for _, tool := range tools {
		if tool == OpenCode {
			return
		}
	}
	t.Fatal("AllTools() does not contain OpenCode")
}

func TestAllTools_Count(t *testing.T) {
	tools := AllTools()
	if len(tools) != 4 {
		t.Fatalf("AllTools() returned %d tools, want 4", len(tools))
	}
}

func TestToolRegistry_OpenCodeEntry(t *testing.T) {
	cfg, ok := toolRegistry[OpenCode]
	if !ok {
		t.Fatal("toolRegistry does not contain OpenCode entry")
	}
	if cfg.PackageDir != "packages/opencode-cli" {
		t.Fatalf("OpenCode PackageDir = %q, want %q", cfg.PackageDir, "packages/opencode-cli")
	}
	if cfg.GenerateScript != "bin/generate.mjs" {
		t.Fatalf("OpenCode GenerateScript = %q, want %q", cfg.GenerateScript, "bin/generate.mjs")
	}
	if cfg.StatusScript != "bin/status.mjs" {
		t.Fatalf("OpenCode StatusScript = %q, want %q", cfg.StatusScript, "bin/status.mjs")
	}
}
