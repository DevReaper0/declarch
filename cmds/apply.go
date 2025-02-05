package cmds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/DevReaper0/declarch/modules"
	"github.com/DevReaper0/declarch/modules/config/ini"
	"github.com/DevReaper0/declarch/parser"
	"github.com/DevReaper0/declarch/utils"
)

// The default tag includes everything without the exclamation mark.
var tags = []string{"+default"}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		configPath, _ = filepath.Abs(configPath)

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
		}

		section, err := parser.ParseFile(configPath)
		if err != nil {
			color.Set(color.FgRed)
			fmt.Print("Error parsing configuration file: ")
			color.Set(color.Bold)
			fmt.Print(configPath)
			color.Set(color.ResetBold)
			fmt.Println(":")
			color.Unset()
			fmt.Fprintln(os.Stderr, err)
			return
		}

		// parserTester(section)
		// return

		if Verify(section) == "" {
			color.Set(color.FgGreen, color.Bold)
			fmt.Println("Configuration is valid.")
			color.Unset()
		} else {
			color.Set(color.FgRed, color.Bold)
			fmt.Println("Configuration is invalid.")
			color.Unset()
			return
		}

		previousSection, _ := parser.Parse("")

		if _, err := os.Stat(configPath + ".prev"); !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error checking for previous configuration file: ")
				color.Set(color.Bold)
				fmt.Print(configPath + ".prev")
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}
			previousSection, err = parser.ParseFile(configPath + ".prev")
			if err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error parsing previous configuration file: ")
				color.Set(color.Bold)
				fmt.Print(configPath + ".prev")
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}

		if err := Apply(section, previousSection); err != nil {
			color.Set(color.FgRed)
			fmt.Print("Error applying configuration: ")
			color.Set(color.Bold)
			fmt.Print(configPath)
			color.Set(color.ResetBold)
			fmt.Println(":")
			color.Unset()
			fmt.Fprintln(os.Stderr, err)
			return
		}

		if _, err := os.Stat(configPath + ".prev"); !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error checking for previous configuration file: ")
				color.Set(color.Bold)
				fmt.Print(configPath + ".prev")
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if err := os.Remove(configPath + ".prev"); err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error removing previous configuration file: ")
				color.Set(color.Bold)
				fmt.Print(configPath + ".prev")
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}
		}
		content := section.Marshal(0)
		if err := os.WriteFile(configPath+".prev", []byte(content), 0o644); err != nil {
			color.Set(color.FgRed)
			fmt.Print("Error creating previous configuration file: ")
			color.Set(color.Bold)
			fmt.Print(configPath + ".prev")
			color.Set(color.ResetBold)
			fmt.Println(":")
			color.Unset()
			fmt.Fprintln(os.Stderr, err)
			return
		}

		color.Set(color.FgGreen, color.Bold)
		fmt.Println("\nConfiguration applied successfully.")
		color.Unset()
	},
}

func Apply(section *parser.Section, previousSection *parser.Section) error {
	modules.PrivilegeEscalationCommand = section.GetFirst("essentials/privilige_escalation", "sudo")
	if modules.PrivilegeEscalationCommand == "su" {
		modules.PrivilegeEscalationCommand = "su -c"
	}

	// Pacman configuration
	if err := configurePacman(section); err != nil {
		return fmt.Errorf("error configuring pacman: %w", err)
	}

	if err := modules.PacmanInstall("base base-devel git"); err != nil {
		return err
	}

	// Temporary fix the non-root commands since user management is not implemented yet.
	// For now, a user must be defined the configuration and the user must already exist on the system.
	// Otherwise, the "nobody" user will be used.
	utils.NormalUser = section.GetFirst("users/user/username", "nobody")

	addedKernels, removedKernels := utils.GetDifferences(getAll(section, "essentials/kernel"), getAll(previousSection, "essentials/kernel"))
	// Installing a kernel before removing any just to be safe
	if len(addedKernels)-1 >= 0 {
		if err := modules.PacmanInstall(addedKernels[len(addedKernels)-1]); err != nil {
			return err
		}
	}
	for i := 0; i < len(removedKernels); i++ {
		if err := modules.PacmanRemove(removedKernels[i]); err != nil {
			return err
		}
	}
	for i := len(addedKernels) - 2; i >= 0; i-- {
		if err := modules.PacmanInstall(addedKernels[i]); err != nil {
			return err
		}
	}

	addedBootloader, removedBootloader := utils.GetDifferences([]string{section.GetFirst("essentials/bootloader", "grub")}, []string{previousSection.GetFirst("essentials/bootloader", "")})
	if len(removedBootloader) > 0 {
		if err := modules.PacmanRemove(removedBootloader[0]); err != nil {
			return err
		}
	}
	if len(addedBootloader) > 0 {
		if err := modules.PacmanInstall(addedBootloader[0]); err != nil {
			return err
		}
	}

	addedPacmanPackages, removedPacmanPackages := utils.GetDifferences(getAll(section, "packages/pacman/package"), getAll(previousSection, "packages/pacman/package"))
	if len(removedPacmanPackages) > 0 {
		if err := modules.PacmanRemove(strings.Join(removedPacmanPackages, " ")); err != nil {
			return err
		}
	}
	if len(addedPacmanPackages) > 0 {
		if err := modules.PacmanInstall(strings.Join(addedPacmanPackages, " ")); err != nil {
			return err
		}
	}

	aurHelper := section.GetFirst("packages/aur/helper", "makepkg")
	addedAurHelper, removedAurHelper := utils.GetDifferences([]string{section.GetFirst("packages/aur/helper", "makepkg")}, []string{previousSection.GetFirst("packages/aur/helper", "")})
	if len(removedAurHelper) > 0 && removedAurHelper[0] != "makepkg" {
		if err := modules.PacmanRemove(removedAurHelper[0]); err != nil {
			return err
		}
	}
	if len(addedAurHelper) > 0 && addedAurHelper[0] != "makepkg" {
		if err := modules.MakepkgInstall(addedAurHelper[0]); err != nil {
			return err
		}
	}

	addedAurPackages, removedAurPackages := utils.GetDifferences(getAll(section, "packages/aur/package"), getAll(previousSection, "packages/aur/package"))
	if len(removedAurPackages) > 0 {
		if err := modules.PacmanRemove(strings.Join(removedAurPackages, " ")); err != nil {
			return err
		}
	}
	if len(addedAurPackages) > 0 {
		if err := modules.AURInstall(aurHelper, strings.Join(addedAurPackages, " ")); err != nil {
			return err
		}
	}

	addedNetworkHandler, removedNetworkHandler := utils.GetDifferences([]string{section.GetFirst("essentials/network_handler", "networkmanager")}, []string{previousSection.GetFirst("essentials/network_handler", "")})
	if len(addedNetworkHandler) > 0 {
		if err := modules.PacmanInstall(addedNetworkHandler[len(addedNetworkHandler)-1]); err != nil {
			return err
		}
	}
	if len(removedNetworkHandler) > 0 {
		if err := modules.PacmanRemove(removedNetworkHandler[0]); err != nil {
			return err
		}
	}

	// TODO

	return nil
}

// transformBooleanOption converts a boolean string ("true"/"false") to its pacman boolean representation.
func transformBooleanOption(value string) string {
	if val, err := strconv.ParseBool(value); err == nil && val {
		return "~BOOL"
	}
	return ""
}

func configurePacman(section *parser.Section) error {
	pacmanConfigPath := "/etc/pacman.conf"
	pacmanParser := ini.NewPacmanParser()
	pacmanPatcher := &ini.Patcher{}

	replaceComments := section.GetFirst("config_parser/replace_comments", "true")
	if val, err := strconv.ParseBool(replaceComments); err == nil && val {
		pacmanPatcher.ReplaceComments = true
	}

	pacmanModifications := map[string]interface{}{}
	addPacmanOption := func(key, value string) {
		if value != "" {
			if _, ok := pacmanModifications["options"]; !ok {
				pacmanModifications["options"] = map[string]interface{}{}
			}
			pacmanModifications["options"].(map[string]interface{})[key] = value
		}
	}

	addPacmanOption("Color", transformBooleanOption(section.GetFirst("packages/pacman/color", "false")))
	addPacmanOption("ParallelDownloads", section.GetFirst("packages/pacman/parallel_downloads", ""))
	addPacmanOption("VerbosePkgLists", transformBooleanOption(section.GetFirst("packages/pacman/verbose_pkg_lists", "false")))
	addPacmanOption("ILoveCandy", transformBooleanOption(section.GetFirst("packages/pacman/i_love_candy", "false")))

	// Add pacman repositories
	repositories := getAllSections(section, "packages/pacman/repository")
	for _, repo := range repositories {
		repoName := repo.GetFirst("name", "")
		if repoName != "" {
			repoModifications := map[string]interface{}{
				"Include": repo.GetFirst("include", ""),
				"Server":  repo.GetFirst("server", ""),
			}
			if _, ok := pacmanModifications[repoName]; !ok {
				pacmanModifications[repoName] = map[string]interface{}{}
			}
			for key, value := range repoModifications {
				if value != "" {
					pacmanModifications[repoName].(map[string]interface{})[key] = value
				}
			}
		}
	}

	if len(pacmanModifications) > 0 {
		if err := pacmanPatcher.Patch(pacmanParser, pacmanConfigPath, pacmanModifications); err != nil {
			return err
		}
	}

	return nil
}

func getAllSections(section *parser.Section, key string) []*parser.Section {
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return []*parser.Section{}
	}

	if len(parts) == 1 {
		if sections, ok := section.Sections[parts[0]]; ok {
			return sections
		}
		return []*parser.Section{}
	}

	subSectionName := parts[0]
	subSectionPath := strings.TrimPrefix(key, subSectionName+"/")

	sections := []*parser.Section{}
	if subSections, ok := section.Sections[subSectionName]; ok {
		for _, subSection := range subSections {
			sections = append(sections, getAllSections(subSection, subSectionPath)...)
		}
	}

	return sections
}

// This function is the same as section.GetAll(), but if an item has spaces, it will be split into multiple items.
// It also supports tags. For example, `package = linux-headers linux-firmware, +bare` needs to be split into `linux-headers` and `linux-firmware`.
func getAll(section *parser.Section, key string) []string {
	items := section.GetAll(key)
	result := []string{}
	for i := 0; i < len(items); i++ {
		included := true

		parts := strings.SplitN(items[i], ",", 2)

		// Check for tagging, e.g., "pkg, +!bare" etc.
		if len(parts) > 1 {
			tagPart := strings.TrimSpace(parts[1])
			linkedTags := strings.Fields(tagPart)

			for _, linkedTag := range linkedTags {
				if !strings.HasPrefix(linkedTag, "+") {
					// TODO: Error in verification
					continue
				}

				tagName := strings.TrimPrefix(linkedTag, "+")
				isRequired := false
				if strings.HasPrefix(tagName, "!") {
					included = false
					tagName = strings.TrimPrefix(tagName, "!")
					isRequired = true
				}

				for _, tag := range tags {
					if tag == "+"+tagName || (!isRequired && tag == "+default") {
						included = true
					} else if tag == "-"+tagName || (!isRequired && tag == "-default") {
						included = false
					}
				}
			}
		} else {
			for _, tag := range tags {
				if tag == "+default" {
					included = true
				} else if tag == "-default" {
					included = false
				}
			}
		}

		if included {
			valuesPart := strings.TrimSpace(parts[0])
			values := strings.Fields(valuesPart)

			result = append(result, values...)
		}
	}
	return result
}

func init() {
	applyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file (default is /etc/declarch/declarch.conf)")
	applyCmd.PersistentFlags().BoolP("bare", "b", false, "Install only essential packages with the +bare tag")

	providedTags := []string{}
	applyCmd.PersistentFlags().StringSliceVar(&providedTags, "tags", []string{}, "List of tags to include/exclude, e.g. '-default +bare'")
	tags = append(tags, providedTags...)

	rootCmd.AddCommand(applyCmd)
}

func parserTester(section *parser.Section) {
	fmt.Println(section.GetFirst("bakery/secrets/password", "!!!!"))
	fmt.Println(section.GetAll("bakery/secrets/password"))
	fmt.Println()
	fmt.Println(section.GetFirst("bakery/employees", "!!!!"))
	fmt.Println(section.GetAll("bakery/employees"))
	fmt.Println()
	fmt.Println(section.GetFirst("cakes/number", "!!!!"))
	fmt.Println(section.GetAll("cakes/number"))
	fmt.Println()
	fmt.Println(section.GetFirst("cakes/colors", "!!!!"))
	fmt.Println(section.GetAll("cakes/colors"))
	fmt.Println()
	fmt.Println(section.GetFirst("bakery/cakes/colors", "!!!!"))
	fmt.Println(section.GetAll("bakery/cakes/colors"))
	fmt.Println()
	fmt.Println(section.GetFirst("add_baker", "!!!!"))
	fmt.Println(section.GetAll("add_baker"))
	fmt.Println()
	fmt.Println(section.GetFirst("abc", "!!!!"))
	fmt.Println(section.GetAll("abc"))
	fmt.Println()
	fmt.Println()
	output := section.Marshal(0)
	fmt.Println(output)
}
