package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func tempNonTTYFile(t *testing.T) (*os.File, error) {
	t.Helper()
	return os.Create(filepath.Join(t.TempDir(), "regular"))
}
