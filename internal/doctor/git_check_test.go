package doctor

import "testing"

func TestCheckGitOnPath(t *testing.T) {
	if err := CheckGitOnPath(); err != nil {
		t.Fatalf("expected git on PATH, got error: %v", err)
	}
}
