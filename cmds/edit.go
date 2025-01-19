package cmds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/DevReaper0/declarch/utils"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		configPath, _ = filepath.Abs(configPath)

		if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
			color.Set(color.FgRed)
			fmt.Print("Configuration file not found: ")
			color.Set(color.Bold)
			fmt.Print(configPath)
			color.Set(color.ResetBold)
			fmt.Println(".")
			color.Unset()
			return
		}

		if err := utils.OpenEditor(configPath); err != nil {
			color.Set(color.FgRed)
			fmt.Println("Error opening editor:")
			color.Unset()
			fmt.Fprintln(os.Stderr, err)
		}
	},
}

func init() {
	editCmd.PersistentFlags().String("config", "/etc/declarch/declarch.conf", "Configuration file (default is /etc/declarch/declarch.conf)")

	rootCmd.AddCommand(editCmd)
}
