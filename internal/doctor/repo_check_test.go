package doctor

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRepoInitialized(t *testing.T) {
	t.Run("existing dir returns nil", func(t *testing.T) {
		tmp := t.TempDir()
		if err := CheckRepoInitialized(tmp); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("missing dir returns error mentioning rice init", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does", "not", "exist")
		err := CheckRepoInitialized(missing)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "rice init") {
			t.Errorf("error should mention `rice init`, got: %v", err)
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("error should mention path %q, got: %v", missing, err)
		}
	})
}
