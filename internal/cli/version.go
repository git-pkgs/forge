package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("forge " + Version)
	},
}

func init() {
	if Version == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			Version = bi.Main.Version
		}
	}
	rootCmd.AddCommand(versionCmd)
}
