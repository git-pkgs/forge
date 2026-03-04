package main

import (
	"os"

	"github.com/git-pkgs/forge/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
