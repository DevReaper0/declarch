package cmds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		configPath, _ = filepath.Abs(configPath)

		// Create file if it doesn't exist
		if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error creating configuration directory: ")
				color.Set(color.Bold)
				fmt.Print(filepath.Dir(configPath))
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}

			if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error creating configuration file: ")
				color.Set(color.Bold)
				fmt.Print(configPath)
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}
		} else if err != nil {
			color.Set(color.FgRed)
			fmt.Print("Error checking configuration file: ")
			color.Set(color.Bold)
			fmt.Print(configPath)
			color.Set(color.ResetBold)
			fmt.Println(":")
			fmt.Fprintln(os.Stderr, err)
			return
		} else {
			color.Set(color.FgRed)
			fmt.Print("Configuration file already exists: ")
			color.Set(color.Bold)
			fmt.Print(configPath)
			color.Set(color.ResetBold)
			fmt.Println(".")
			color.Unset()
			return
		}

		color.Set(color.FgGreen)
		fmt.Print("Configuration file created successfully: ")
		color.Set(color.Bold)
		fmt.Print(configPath)
		color.Set(color.ResetBold)
		fmt.Println(".")
		color.Unset()
	},
}

func init() {
	initCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file (default is /etc/declarch/declarch.conf)")

	rootCmd.AddCommand(initCmd)
}
