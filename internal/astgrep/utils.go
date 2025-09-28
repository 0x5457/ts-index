package astgrep

import (
	"os"
)

// writeFile writes content to a file
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// removeFile removes a file
func removeFile(path string) error {
	return os.Remove(path)
}
