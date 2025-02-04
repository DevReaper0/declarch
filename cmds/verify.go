package cmds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

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
	if v := VerifyString(section.GetFirst("essentials/privilige_escalation", "sudo"), "sudo", "doas", "pkexec", "su"); v != "" {
		return v
	}

	// Validate pacman configuration options
	if v := VerifyPacmanConfig(section); v != "" {
		return v
	}

	for _, item := range section.GetAll("packages/pacman/package") {
		if v := VerifyTags(item); v != "" {
			return v
		}
	}
	for _, item := range section.GetAll("packages/aur/package") {
		if v := VerifyTags(item); v != "" {
			return v
		}
	}

	return ""
}

func VerifyPacmanConfig(section *parser.Section) string {
	if v := VerifyString(section.GetFirst("packages/pacman/color", "true"), "true", "false"); v != "" {
		return v
	}
	if v := VerifyString(section.GetFirst("packages/pacman/verbose_pkg_lists", "false"), "true", "false"); v != "" {
		return v
	}
	if v := VerifyString(section.GetFirst("packages/pacman/i_love_candy", "false"), "true", "false"); v != "" {
		return v
	}
	if parallelDownloads := section.GetFirst("packages/pacman/parallel_downloads", ""); parallelDownloads != "" {
		if _, err := strconv.Atoi(parallelDownloads); err != nil {
			return fmt.Sprintf("Invalid value for parallel_downloads: %s", parallelDownloads)
		}
	}
	return ""
}

func VerifyTags(packageEntry string) string {
	parts := strings.SplitN(packageEntry, ",", 2)
	if len(parts) <= 1 {
		return ""
	}

	tagPart := strings.TrimSpace(parts[1])
	for _, tag := range strings.Fields(tagPart) {
		if v := VerifyTag(tag); v != "" {
			return v
		}
	}
	return ""
}

func VerifyTag(tag string) string {
	if !strings.HasPrefix(tag, "+") {
		return fmt.Sprintf("Tag '%s' must start with '+' character", tag)
	}

	tagName := strings.TrimPrefix(tag, "+")
	if strings.HasPrefix(tagName, "!") {
		tagName = strings.TrimPrefix(tagName, "!")
	}

	if len(tagName) == 0 {
		return "Tag name cannot be empty"
	}

	for _, char := range tagName {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && char != '_' && char != '-' {
			return fmt.Sprintf("Tag name '%s' contains invalid character '%c'", tagName, char)
		}
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
	verifyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file (default is /etc/declarch/declarch.conf)")

	rootCmd.AddCommand(verifyCmd)
}
