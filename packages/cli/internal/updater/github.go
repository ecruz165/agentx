package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/agentx-labs/agentx/internal/branding"
)

const (
	githubAPIBase = "https://api.github.com"
)

// CheckLatestVersion fetches the latest release from GitHub.
func (u *Updater) CheckLatestVersion() (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, branding.GitHubRepo())
	return u.fetchRelease(url)
}

// CheckSpecificVersion fetches a release by tag from GitHub.
func (u *Updater) CheckSpecificVersion(tag string) (*Release, error) {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", githubAPIBase, branding.GitHubRepo(), tag)
	return u.fetchRelease(url)
}

func (u *Updater) fetchRelease(url string) (*Release, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "agentx-updater")

	// Support optional GitHub token for higher rate limits.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release not found")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("GitHub API rate limit exceeded. Set GITHUB_TOKEN for higher limits")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("parsing release JSON: %w", err)
	}

	release.TagName = release.Version

	// If a mirror is configured, rewrite asset download URLs.
	if u.mirror != "" {
		for i := range release.Assets {
			release.Assets[i].DownloadURL = strings.TrimRight(u.mirror, "/") + "/" + release.Assets[i].Name
		}
	}

	return &release, nil
}
