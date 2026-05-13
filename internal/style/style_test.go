package style

import "testing"

func TestPlainToggle(t *testing.T) {
	t.Cleanup(func() { SetPlain(false) })

	if Plain() {
		t.Fatal("expected Plain() == false initially")
	}
	SetPlain(true)
	if !Plain() {
		t.Fatal("expected Plain() == true after SetPlain(true)")
	}
	SetPlain(false)
	if Plain() {
		t.Fatal("expected Plain() == false after SetPlain(false)")
	}
}
