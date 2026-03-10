package cli

import (
	"errors"
	"testing"

	forges "github.com/git-pkgs/forge"
)

func TestNotSupportedWrapsError(t *testing.T) {
	err := notSupported(forges.ErrNotSupported, "CI pipelines")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	want := "CI pipelines is not supported by this forge"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestNotSupportedPassesThrough(t *testing.T) {
	original := errors.New("connection refused")
	err := notSupported(original, "CI pipelines")
	if err != original {
		t.Errorf("expected original error %q, got %q", original, err)
	}
}
