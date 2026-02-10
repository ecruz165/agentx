// Package platform provides cross-platform filesystem operations including
// symlink creation and permission management. On Unix systems it uses native
// symlinks and chmod directly. On Windows it falls back to file copying with
// a .target sidecar when developer mode symlinks are unavailable.
package platform