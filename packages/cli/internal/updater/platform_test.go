package updater

import (
	"runtime"
	"strings"
	"testing"
)

func TestArchiveName(t *testing.T) {
	name := ArchiveName()
	if !strings.Contains(name, runtime.GOOS) {
		t.Errorf("ArchiveName() = %q, does not contain GOOS %q", name, runtime.GOOS)
	}
	if !strings.Contains(name, runtime.GOARCH) {
		t.Errorf("ArchiveName() = %q, does not contain GOARCH %q", name, runtime.GOARCH)
	}
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(name, ".zip") {
			t.Errorf("expected .zip suffix on Windows, got %q", name)
		}
	} else {
		if !strings.HasSuffix(name, ".tar.gz") {
			t.Errorf("expected .tar.gz suffix, got %q", name)
		}
	}
}

func TestSelectAssetForPlatform(t *testing.T) {
	expected := ArchiveName()
	assets := []Asset{
		{Name: "agentx_linux_amd64.tar.gz", DownloadURL: "https://example.com/linux"},
		{Name: "agentx_darwin_arm64.tar.gz", DownloadURL: "https://example.com/darwin-arm"},
		{Name: "agentx_darwin_amd64.tar.gz", DownloadURL: "https://example.com/darwin-amd"},
		{Name: "agentx_windows_amd64.zip", DownloadURL: "https://example.com/windows"},
		{Name: "checksums.txt", DownloadURL: "https://example.com/checksums"},
	}

	asset, err := SelectAssetForPlatform(assets)
	if err != nil {
		t.Fatalf("SelectAssetForPlatform failed: %v", err)
	}
	if asset.Name != expected {
		t.Errorf("selected asset %q, expected %q", asset.Name, expected)
	}
}

func TestSelectAssetForPlatform_NoMatch(t *testing.T) {
	assets := []Asset{
		{Name: "agentx_freebsd_amd64.tar.gz", DownloadURL: "https://example.com/freebsd"},
	}

	_, err := SelectAssetForPlatform(assets)
	if err == nil {
		t.Error("expected error for no matching asset")
	}
}

func TestSelectAssetForPlatform_FlexibleMatch(t *testing.T) {
	// Simulate a slightly different naming convention.
	pattern := runtime.GOOS + "_" + runtime.GOARCH
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	flexName := "agentx_v1.0.0_" + pattern + ext

	assets := []Asset{
		{Name: flexName, DownloadURL: "https://example.com/flex"},
	}

	asset, err := SelectAssetForPlatform(assets)
	if err != nil {
		t.Fatalf("SelectAssetForPlatform flexible match failed: %v", err)
	}
	if asset.Name != flexName {
		t.Errorf("selected %q, expected %q", asset.Name, flexName)
	}
}
