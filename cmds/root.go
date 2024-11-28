package cmds

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "declarch",
	Short: "DeclArch is a tool for declaratively managing an Arch Linux system",
	Long:  "DeclArch is a tool for declaratively managing an Arch Linux system",
}

func Execute() {
	rootCmd.Execute()
}
