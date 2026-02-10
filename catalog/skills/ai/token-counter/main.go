// token-counter -- AgentX Skill (Go)
// Counts tokens in text using tiktoken for a given model encoding.
//
// This skill is self-contained (no external CLI dependency). It compiles
// to a standalone binary and follows the AgentX registry pattern.
//
// Usage: token-counter --text="Hello world" [--model=gpt-4]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

const (
	skillTopic = "ai"
	skillName  = "token-counter"
)

type output struct {
	Timestamp string      `json:"timestamp"`
	Skill     string      `json:"skill"`
	Status    string      `json:"status"`
	Data      interface{} `json:"data"`
}

type tokenResult struct {
	Model      string `json:"model"`
	TokenCount int    `json:"token_count"`
	TextLength int    `json:"text_length"`
}

func userdataRoot() string {
	if v := os.Getenv("AGENTX_USERDATA"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".agentx", "userdata")
}

func saveOutput(data interface{}) error {
	dir := filepath.Join(userdataRoot(), "skills", skillTopic, skillName, "output")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling output: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "latest.json"), payload, 0o644)
}

func main() {
	text := flag.String("text", "", "Text to count tokens for")
	model := flag.String("model", "gpt-4", "Model name to determine encoding")
	flag.Parse()

	if *text == "" {
		fmt.Fprintln(os.Stderr, "error: --text is required")
		flag.Usage()
		os.Exit(1)
	}

	enc, err := tiktoken.EncodingForModel(*model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get encoding for model %q: %v\n", *model, err)
		os.Exit(1)
	}

	tokens := enc.Encode(*text, nil, nil)

	result := output{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Skill:     skillName,
		Status:    "ok",
		Data: tokenResult{
			Model:      *model,
			TokenCount: len(tokens),
			TextLength: len(*text),
		},
	}

	if err := saveOutput(result); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save output: %v\n", err)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}
