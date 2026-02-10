package platform

import (
	"os"
	"runtime"
)

// Chmod sets file permissions. On Windows this is a no-op because Windows
// does not support Unix-style permission bits.
func Chmod(path string, mode os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, mode)
}
