package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadBinary downloads the appropriate asset for the current platform.
// Returns the path to the downloaded archive file.
func (u *Updater) DownloadBinary(release *Release, destDir string) (string, error) {
	asset, err := SelectAssetForPlatform(release.Assets)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(destDir, asset.Name)

	req, err := http.NewRequest("GET", asset.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "agentx-updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", asset.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("creating download file: %w", err)
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	lastPercent := -1

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return "", fmt.Errorf("writing download: %w", writeErr)
			}
			downloaded += int64(n)
			if total > 0 {
				percent := int(downloaded * 100 / total)
				if percent != lastPercent {
					fmt.Fprintf(os.Stderr, "\rDownloading... %d%%", percent)
					lastPercent = percent
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("reading download stream: %w", readErr)
		}
	}
	if total > 0 {
		fmt.Fprintln(os.Stderr)
	}

	return destPath, nil
}

// VerifyChecksum downloads checksums.txt from the release and verifies the archive.
func (u *Updater) VerifyChecksum(release *Release, archivePath string) error {
	// Find checksums.txt asset.
	var checksumAsset *Asset
	for i := range release.Assets {
		if release.Assets[i].Name == "checksums.txt" {
			checksumAsset = &release.Assets[i]
			break
		}
	}
	if checksumAsset == nil {
		return fmt.Errorf("checksums.txt not found in release assets")
	}

	// Download checksums.
	req, err := http.NewRequest("GET", checksumAsset.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("creating checksum request: %w", err)
	}
	req.Header.Set("User-Agent", "agentx-updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	// Parse checksums.txt: each line is "sha256  filename".
	archiveName := filepath.Base(archivePath)
	expectedHash := ""
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == archiveName {
			expectedHash = parts[0]
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s in checksums.txt", archiveName)
	}

	// Compute actual hash.
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// ExtractBinary extracts the agentx binary from a tar.gz or zip archive.
// Returns the path to the extracted binary.
func ExtractBinary(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, destDir)
	}
	return extractFromTarGz(archivePath, destDir)
}

func extractFromTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar entry: %w", err)
		}

		baseName := filepath.Base(hdr.Name)
		if baseName == "agentx" || baseName == "agentx.exe" {
			destPath := filepath.Join(destDir, baseName)
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return "", fmt.Errorf("creating binary file: %w", err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", fmt.Errorf("extracting binary: %w", err)
			}
			out.Close()
			return destPath, nil
		}
	}

	return "", fmt.Errorf("agentx binary not found in archive")
}

func extractFromZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening zip archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if baseName == "agentx" || baseName == "agentx.exe" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("opening zip entry: %w", err)
			}

			destPath := filepath.Join(destDir, baseName)
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				rc.Close()
				return "", fmt.Errorf("creating binary file: %w", err)
			}

			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				rc.Close()
				return "", fmt.Errorf("extracting binary: %w", err)
			}
			out.Close()
			rc.Close()
			return destPath, nil
		}
	}

	return "", fmt.Errorf("agentx binary not found in zip archive")
}
