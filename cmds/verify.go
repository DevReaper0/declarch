package cmds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/DevReaper0/declarch/parser"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify configuration",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		configPath, _ = filepath.Abs(configPath)

		section, err := parser.ParseFile(configPath)
		if err != nil {
			color.Set(color.FgRed)
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Print("Configuration file not found: ")
				color.Set(color.Bold)
				fmt.Print(configPath)
				color.Set(color.ResetBold)
				fmt.Println(".")
				color.Unset()
				return
			}
			fmt.Print("Error parsing configuration file: ")
			color.Set(color.Bold)
			fmt.Print(configPath)
			color.Set(color.ResetBold)
			fmt.Println(":")
			color.Unset()
			fmt.Fprintln(os.Stderr, err)
			return
		}

		if Verify(section) == "" {
			color.Set(color.FgGreen, color.Bold)
			fmt.Println("Configuration is valid.")
		} else {
			color.Set(color.FgRed, color.Bold)
			fmt.Println("Configuration is invalid.")
		}
		color.Unset()
	},
}

func Verify(section *parser.Section) string {
	if v := VerifyString(section.GetFirst("essentials/privilige_escalation", "sudo"), "sudo", "doas"); v != "" {
		return v
	}

	return ""
}

func VerifyString(value string, allowedValues ...string) string {
	if slices.Contains(allowedValues, value) {
		return ""
	}
	return fmt.Sprintf("Value '%s' is not allowed. Allowed values are: %s", value, strings.Join(allowedValues, ", "))
}

func init() {
	verifyCmd.PersistentFlags().String("config", "/etc/declarch/declarch.conf", "Configuration file (default is /etc/declarch/declarch.conf)")

	rootCmd.AddCommand(verifyCmd)
}
