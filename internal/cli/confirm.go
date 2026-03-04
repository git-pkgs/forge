package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// confirm asks the user for yes/no confirmation. It returns an error if
// stdin is not a terminal (to prevent piped input from silently bypassing
// the prompt) or if the user does not confirm.
func confirm(prompt string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("refusing destructive action: stdin is not a terminal (use --yes to skip confirmation)")
	}

	fmt.Fprint(os.Stderr, prompt)
	var answer string
	_, _ = fmt.Scanln(&answer)
	if strings.ToLower(answer) != "y" {
		return errors.New("cancelled")
	}
	return nil
}
