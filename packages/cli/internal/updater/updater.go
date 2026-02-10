package updater

import (
	"net/http"
	"time"
)

// Release represents a GitHub release.
type Release struct {
	Version   string    `json:"tag_name"`
	TagName   string    `json:"-"`
	Assets    []Asset   `json:"assets"`
	Published time.Time `json:"published_at"`
	HTMLURL   string    `json:"html_url"`
}

// Asset represents a downloadable file attached to a release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Updater provides self-update functionality.
type Updater struct {
	currentVersion string
	httpClient     *http.Client
	mirror         string
}

// Option configures an Updater.
type Option func(*Updater)

// WithHTTPClient sets a custom HTTP client (useful for testing).
func WithHTTPClient(c *http.Client) Option {
	return func(u *Updater) {
		u.httpClient = c
	}
}

// WithMirror sets a mirror URL for downloading releases.
func WithMirror(mirror string) Option {
	return func(u *Updater) {
		u.mirror = mirror
	}
}

// New creates an Updater with the given current version and options.
func New(currentVersion string, opts ...Option) *Updater {
	u := &Updater{
		currentVersion: currentVersion,
		httpClient:     http.DefaultClient,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// CurrentVersion returns the version this updater was created with.
func (u *Updater) CurrentVersion() string {
	return u.currentVersion
}
