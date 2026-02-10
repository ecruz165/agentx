package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// createTestTarGz creates a tar.gz archive containing a fake "agentx" binary.
func createTestTarGz(t *testing.T, binaryContent []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "agentx",
		Mode: 0755,
		Size: int64(len(binaryContent)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binaryContent); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestDownloadBinary(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho test")
	archiveData := createTestTarGz(t, binaryContent)

	archiveName := ArchiveName()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(archiveData)))
		w.Write(archiveData)
	}))
	defer server.Close()

	u := New("1.0.0", WithHTTPClient(server.Client()))

	release := &Release{
		Version: "v1.1.0",
		Assets: []Asset{
			{Name: archiveName, DownloadURL: server.URL + "/" + archiveName},
		},
	}

	destDir := t.TempDir()
	archivePath, err := u.DownloadBinary(release, destDir)
	if err != nil {
		t.Fatalf("DownloadBinary failed: %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("downloaded file does not exist: %v", err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	binaryContent := []byte("fake binary content")
	archiveData := createTestTarGz(t, binaryContent)

	// Compute checksum.
	h := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(h[:])

	archiveName := ArchiveName()
	checksumContent := fmt.Sprintf("%s  %s\n", checksum, archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumContent))
	}))
	defer server.Close()

	u := New("1.0.0", WithHTTPClient(server.Client()))

	release := &Release{
		Assets: []Asset{
			{Name: "checksums.txt", DownloadURL: server.URL + "/checksums.txt"},
		},
	}

	// Write archive to temp file.
	tmp := t.TempDir()
	archivePath := filepath.Join(tmp, archiveName)
	os.WriteFile(archivePath, archiveData, 0644)

	err := u.VerifyChecksum(release, archivePath)
	if err != nil {
		t.Fatalf("VerifyChecksum failed: %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	archiveName := ArchiveName()
	checksumContent := fmt.Sprintf("%s  %s\n", "0000000000000000000000000000000000000000000000000000000000000000", archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksumContent))
	}))
	defer server.Close()

	u := New("1.0.0", WithHTTPClient(server.Client()))

	release := &Release{
		Assets: []Asset{
			{Name: "checksums.txt", DownloadURL: server.URL + "/checksums.txt"},
		},
	}

	tmp := t.TempDir()
	archivePath := filepath.Join(tmp, archiveName)
	os.WriteFile(archivePath, []byte("different content"), 0644)

	err := u.VerifyChecksum(release, archivePath)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho extracted")
	archiveData := createTestTarGz(t, binaryContent)

	tmp := t.TempDir()
	archivePath := filepath.Join(tmp, "agentx.tar.gz")
	os.WriteFile(archivePath, archiveData, 0644)

	binPath, err := ExtractBinary(archivePath, tmp)
	if err != nil {
		t.Fatalf("ExtractBinary failed: %v", err)
	}

	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(data) != string(binaryContent) {
		t.Errorf("extracted content mismatch")
	}

	// Check executable permission on non-Windows.
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(binPath)
		if info.Mode().Perm()&0111 == 0 {
			t.Error("extracted binary is not executable")
		}
	}
}

func TestVerifyChecksum_MissingAsset(t *testing.T) {
	u := New("1.0.0")
	release := &Release{
		Assets: []Asset{
			{Name: "agentx_darwin_arm64.tar.gz", DownloadURL: "https://example.com/file"},
		},
	}
	err := u.VerifyChecksum(release, "/tmp/some-archive.tar.gz")
	if err == nil {
		t.Error("expected error for missing checksums.txt asset")
	}
}
