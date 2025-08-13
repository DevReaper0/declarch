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
	privilegeEscalation := section.GetFirst("essentials/privilige_escalation", "sudo")
	if !slices.Contains([]string{"sudo", "doas", "pkexec", "su"}, privilegeEscalation) {
		return fmt.Sprintf("Value '%s' is not allowed for privilege escalation. Allowed values are: sudo, doas, pkexec, su", privilegeEscalation)
	}

	if v := verifyUsers(section); v != "" {
		return v
	}

	if v := verifyPacman(section); v != "" {
		return v
	}

	if v := verifyAUR(section); v != "" {
		return v
	}

	if v := verifyFlatpak(section); v != "" {
		return v
	}

	return ""
}

func verifyUsers(section *parser.Section) string {
	userSections := getAllSections(section, "users/user")
	for _, userSection := range userSections {
		username := userSection.GetFirst("username", "")
		if username == "" {
			return "User section missing required 'username' field"
		}

		createHome := userSection.GetFirst("create_home", "true")
		if _, err := strconv.ParseBool(createHome); err != nil {
			return fmt.Sprintf("User '%s': invalid value for 'create_home': %s", username, createHome)
		}
	}

	hookSections := getAllSections(section, "users/hook")
	for _, hookSection := range hookSections {
		username := hookSection.GetFirst("user", "")
		if username == "" {
			return "User hook section missing required 'user' field"
		}

		if v := verifyHook(hookSection, "create", "delete", fmt.Sprintf("for user '%s'", username)); v != "" {
			return v
		}
	}

	return ""
}

func verifyPacman(section *parser.Section) string {
	color := section.GetFirst("packages/pacman/color", "true")
	if _, err := strconv.ParseBool(color); err != nil {
		return fmt.Sprintf("Invalid value for packages/pacman/color: %s", color)
	}

	verbosePkgLists := section.GetFirst("packages/pacman/verbose_pkg_lists", "false")
	if _, err := strconv.ParseBool(verbosePkgLists); err != nil {
		return fmt.Sprintf("Invalid value for packages/pacman/verbose_pkg_lists: %s", verbosePkgLists)
	}

	iLoveCandy := section.GetFirst("packages/pacman/i_love_candy", "false")
	if _, err := strconv.ParseBool(iLoveCandy); err != nil {
		return fmt.Sprintf("Invalid value for packages/pacman/i_love_candy: %s", iLoveCandy)
	}

	if parallelDownloads := section.GetFirst("packages/pacman/parallel_downloads", ""); parallelDownloads != "" {
		if _, err := strconv.Atoi(parallelDownloads); err != nil {
			return fmt.Sprintf("Invalid value for parallel_downloads: %s", parallelDownloads)
		}
	}

	for _, item := range section.GetAll("packages/pacman/package") {
		if v := VerifyTags(item); v != "" {
			return v
		}
	}

	hookSections := getAllSections(section, "packages/pacman/hook")
	for _, hookSection := range hookSections {
		pkgName := hookSection.GetFirst("package", "")
		if pkgName == "" {
			return "Pacman hook section missing required 'package' field"
		}

		if v := verifyHook(hookSection, "install", "remove", fmt.Sprintf("for Pacman package '%s'", pkgName)); v != "" {
			return v
		}
	}

	return ""
}

func verifyAUR(section *parser.Section) string {
	for _, item := range section.GetAll("packages/aur/package") {
		if v := VerifyTags(item); v != "" {
			return v
		}
	}

	hookSections := getAllSections(section, "packages/aur/hook")
	for _, hookSection := range hookSections {
		pkgName := hookSection.GetFirst("package", "")
		if pkgName == "" {
			return "AUR hook section missing required 'package' field"
		}

		if v := verifyHook(hookSection, "install", "remove", fmt.Sprintf("for AUR package '%s'", pkgName)); v != "" {
			return v
		}
	}

	return ""
}

func verifyFlatpak(section *parser.Section) string {
	remoteSections := getAllSections(section, "packages/flatpak/remote")
	for _, remote := range remoteSections {
		name := remote.GetFirst("name", "")
		if name == "" {
			return "Flatpak remote section missing required 'name' field"
		}

		url := remote.GetFirst("url", "")
		if url == "" {
			return fmt.Sprintf("Flatpak remote '%s' missing required 'url' field", name)
		}

		userInstallation := remote.GetFirst("user_installation", "false")
		if _, err := strconv.ParseBool(userInstallation); err != nil {
			return fmt.Sprintf("Flatpak remote '%s': invalid value for 'user_installation': %s", name, userInstallation)
		}

		disable := remote.GetFirst("disable", "false")
		if _, err := strconv.ParseBool(disable); err != nil {
			return fmt.Sprintf("Flatpak remote '%s': invalid value for 'disable': %s", name, disable)
		}
	}

	for _, item := range section.GetAll("packages/flatpak/package") {
		if v := VerifyTags(item); v != "" {
			return v
		}
	}

	packageSections := getAllSections(section, "packages/flatpak/package")
	for _, pkg := range packageSections {
		name := pkg.GetFirst("name", "")
		if name == "" {
			return "Flatpak package section missing required 'name' field"
		}

		userInstallation := pkg.GetFirst("user_installation", "false")
		if _, err := strconv.ParseBool(userInstallation); err != nil {
			return fmt.Sprintf("Flatpak package '%s': invalid value for 'user_installation': %s", name, userInstallation)
		}
	}

	hookSections := getAllSections(section, "packages/flatpak/hook")
	for _, hookSection := range hookSections {
		pkgName := hookSection.GetFirst("package", "")
		if pkgName == "" {
			return "Flatpak hook section missing required 'package' field"
		}

		if v := verifyHook(hookSection, "install", "remove", fmt.Sprintf("for Flatpak package '%s'", pkgName)); v != "" {
			return v
		}
	}

	return ""
}

func verifyHook(section *parser.Section, additionTerm, removalTerm string, context string) string {
	forValue := section.GetFirst("for", additionTerm)
	if forValue != additionTerm && forValue != removalTerm {
		return fmt.Sprintf("Invalid hook %s: 'for' value must be '%s' or '%s', got '%s'",
			context, additionTerm, removalTerm, forValue)
	}

	when := section.GetFirst("when", "after")
	if when != "before" && when != "after" {
		return fmt.Sprintf("Invalid hook %s: 'when' value must be 'before' or 'after', got '%s'",
			context, when)
	}

	run := section.GetFirst("run", "")
	if run == "" {
		return fmt.Sprintf("Invalid hook %s: 'run' field is required", context)
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
	tagName = strings.TrimPrefix(tagName, "!")

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

func init() {
	verifyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file")

	rootCmd.AddCommand(verifyCmd)
}