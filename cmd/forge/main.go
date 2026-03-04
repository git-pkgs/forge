package main

import (
	"os"

	"github.com/git-pkgs/forges/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
