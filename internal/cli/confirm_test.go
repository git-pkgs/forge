package cli

import (
	"os"
	"strings"
	"testing"
)

func TestConfirmNonTerminal(t *testing.T) {
	// When stdin is a pipe (not a terminal), confirm should refuse.
	// In test context, stdin is not a terminal.
	err := confirm("Delete? [y/N] ")
	if err == nil {
		t.Fatal("expected error for non-terminal stdin")
	}
	if !strings.Contains(err.Error(), "not a terminal") {
		t.Errorf("expected 'not a terminal' in error, got: %s", err)
	}
}

func TestConfirmWithPipedNo(t *testing.T) {
	// Even with a piped "n", it should fail because stdin isn't a terminal.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.WriteString("n\n")
	w.Close()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	err = confirm("Delete? [y/N] ")
	if err == nil {
		t.Fatal("expected error for piped stdin")
	}
	if !strings.Contains(err.Error(), "not a terminal") {
		t.Errorf("expected 'not a terminal' in error, got: %s", err)
	}
}
