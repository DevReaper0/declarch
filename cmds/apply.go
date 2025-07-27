package cmds

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
var tags []string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		configPath, _ = filepath.Abs(configPath)

		if bare, _ := cmd.PersistentFlags().GetBool("bare"); bare {
			tags = append(tags, "-default", "+bare")
		}

		tags = append([]string{"+default"}, tags...)

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

		if up, _ := cmd.PersistentFlags().GetBool("upgrade"); up {
			if err := Upgrade(section); err != nil {
				color.Set(color.FgRed)
				fmt.Print("Error upgrading system: ")
				color.Set(color.Bold)
				fmt.Print(configPath)
				color.Set(color.ResetBold)
				fmt.Println(":")
				color.Unset()
				fmt.Fprintln(os.Stderr, err)
				return
			}
			color.Set(color.FgGreen, color.Bold)
			fmt.Println("System upgraded successfully.")
			color.Unset()
			return
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

	// Temporary fix for the non-root commands since user management is not implemented yet.
	// For now, a user must be defined the configuration and the user must already exist on the system.
	// Otherwise, the "nobody" user will be used.
	utils.NormalUser = section.GetFirst("users/user/username", "nobody")

	packageCommandHooks := getAllSections(section, "packages/command_hooks/hook")

	aurHelper := section.GetFirst("packages/aur/helper", "makepkg")
	aurInstaller := func(name string) error { return modules.AURInstall(aurHelper, name) }

	// Handle kernels
	kernelList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedKernels, removedKernels := utils.GetDifferences(getAllKernels(section, "essentials/kernel"), getAllKernels(previousSection, "essentials/kernel"))
	// Installing a kernel before removing any just to be safe
	if len(addedKernels) > 0 {
		pkgName := addedKernels[len(addedKernels)-1]
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		kernelList.Add(pkg)
	}
	if err := kernelList.Install(); err != nil {
		return err
	}
	kernelList.Clear()
	for _, pkgName := range removedKernels {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "remove" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		kernelList.Add(pkg)
	}
	if err := kernelList.Remove(); err != nil {
		return err
	}
	kernelList.Clear()
	for i := len(addedKernels) - 2; i >= 0; i-- {
		pkgName := addedKernels[i]
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		kernelList.Add(pkg)
	}
	if err := kernelList.Install(); err != nil {
		return err
	}

	// Handle bootloader
	bootloaderList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedBootloader, removedBootloader := utils.GetDifferences([]string{section.GetFirst("essentials/bootloader", "grub")}, []string{previousSection.GetFirst("essentials/bootloader", "")})
	for _, pkgName := range removedBootloader {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "remove" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		bootloaderList.Add(pkg)
	}
	if err := bootloaderList.Remove(); err != nil {
		return err
	}
	bootloaderList.Clear()
	for _, pkgName := range addedBootloader {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		bootloaderList.Add(pkg)
	}
	// TODO: Some way to disable installing efibootmgr if the bootloader doesn't use it??
	if len(addedBootloader) > 0 {
		pkgName := "efibootmgr"
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		bootloaderList.Add(pkg)
	}
	if err := bootloaderList.Install(); err != nil {
		return err
	}

	// Handle pacman packages
	pacmanList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedPacmanPackages, removedPacmanPackages := utils.GetDifferences(getAll(section, "packages/pacman/package"), getAll(previousSection, "packages/pacman/package"))
	for _, pkgName := range removedPacmanPackages {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "remove" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		pacmanList.Add(pkg)
	}
	if err := pacmanList.Remove(); err != nil {
		return err
	}
	pacmanList.Clear()
	for _, pkgName := range addedPacmanPackages {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		pacmanList.Add(pkg)
	}
	if err := pacmanList.Install(); err != nil {
		return err
	}

	// Handle AUR packages
	aurList := modules.NewPackageList(aurInstaller, modules.PacmanRemove)
	addedAurPackages, removedAurPackages := utils.GetDifferences(getAll(section, "packages/aur/package"), getAll(previousSection, "packages/aur/package"))
	for _, pkgName := range removedAurPackages {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "remove" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		aurList.Add(pkg)
	}
	if err := aurList.Remove(); err != nil {
		return err
	}
	aurList.Clear()
	for _, pkgName := range addedAurPackages {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		aurList.Add(pkg)
	}
	if err := aurList.Install(); err != nil {
		return err
	}

	// Handle network handler
	networkHandlerList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedNetworkHandler, removedNetworkHandler := utils.GetDifferences([]string{section.GetFirst("essentials/network_handler", "networkmanager")}, []string{previousSection.GetFirst("essentials/network_handler", "")})
	for _, pkgName := range removedNetworkHandler {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "remove" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		networkHandlerList.Add(pkg)
	}
	if err := networkHandlerList.Remove(); err != nil {
		return err
	}
	networkHandlerList.Clear()
	for _, pkgName := range addedNetworkHandler {
		pkg := modules.NewPackage(pkgName)
		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		networkHandlerList.Add(pkg)
	}
	if err := networkHandlerList.Install(); err != nil {
		return err
	}

	// TODO

	return nil
}

func Upgrade(section *parser.Section) error {
	modules.PrivilegeEscalationCommand = section.GetFirst("essentials/privilige_escalation", "sudo")
	if modules.PrivilegeEscalationCommand == "su" {
		modules.PrivilegeEscalationCommand = "su -c"
	}

	// Temporary fix for the non-root commands since user management is not implemented yet.
	// For now, a user must be defined the configuration and the user must already exist on the system.
	// Otherwise, the "nobody" user will be used.
	utils.NormalUser = section.GetFirst("users/user/username", "nobody")

	// Check which sections are actually configured
	pacmanConfigured := false
	aurConfigured := false

	// Check if pacman section exists and has at least one package
	if packageSections, ok := section.Sections["packages"]; ok {
		for _, pkgSection := range packageSections {
			if _, hasPacman := pkgSection.Sections["pacman"]; hasPacman {
				pacmanConfigured = true
				break
			}
		}
	}

	// Check if AUR section exists and has a helper configured
	aurHelper := section.GetFirst("packages/aur/helper", "")
	aurPackages := getAll(section, "packages/aur/package")
	aurConfigured = aurHelper != "" && len(aurPackages) > 0

	availableUpgrades := []string{}
	if pacmanConfigured {
		availableUpgrades = append(availableUpgrades, "pacman:system packages via Pacman")
	}
	if aurConfigured {
		availableUpgrades = append(availableUpgrades, "aur:AUR packages via "+aurHelper)
	}

	// If no package managers are configured, inform the user
	if len(availableUpgrades) == 0 {
		color.Set(color.FgYellow)
		fmt.Println("No package managers configured for upgrade.")
		color.Unset()
		return nil
	}

	toUpgrade := confirmUpgradeAll(availableUpgrades)

	if slices.Contains(toUpgrade, "pacman") {
		color.Set(color.FgCyan)
		fmt.Println("Upgrading system packages via Pacman...")
		color.Unset()

		if err := modules.PacmanSystemUpgrade(); err != nil {
			return err
		}
	}

	if slices.Contains(toUpgrade, "aur") {
		color.Set(color.FgCyan)
		fmt.Println("Upgrading AUR packages via " + aurHelper + "...")
		color.Unset()

		if err := modules.PacmanWrapperSystemUpgrade(aurHelper); err != nil {
			return err
		}
	}

	return nil
}

// confirmUpgrade asks the user whether they want to upgrade a specific package manager
// Returns true if the user confirms or doesn't provide input (default yes)
func confirmUpgrade(packageType string) bool {
	color.Set(color.FgCyan)
	fmt.Printf("Do you want to upgrade %s? [Y/n] ", packageType)
	color.Unset()

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "" || response == "y" || response == "yes"
}

// confirmUpgradeAll asks the user whether they want to upgrade all available tools
func confirmUpgradeAll(availableUpgrades []string) []string {
	toUpgrade := []string{}

	if len(availableUpgrades) > 1 {
		color.Set(color.FgCyan)
		fmt.Print("Do you want to upgrade all available tools (")
		for i, upgrade := range availableUpgrades {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(strings.SplitN(upgrade, ":", 2)[1])
		}
		fmt.Print(")? [Y/n] ")
		color.Unset()

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "" || response == "y" || response == "yes" {
			for _, upgrade := range availableUpgrades {
				toUpgrade = append(toUpgrade, strings.SplitN(upgrade, ":", 2)[0])
			}
			return toUpgrade
		}
	}

	for _, upgrade := range availableUpgrades {
		upgradeName, upgradeDesc, _ := strings.Cut(upgrade, ":")
		if confirmUpgrade(upgradeDesc) {
			toUpgrade = append(toUpgrade, upgradeName)
		}
	}
	return toUpgrade
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

	builtinRepositories := []string{
		"core-testing",
		"core",
		"extra-testing",
		"extra",
		"multilib-testing",
		"multilib",
	}

	// Add pacman repositories
	repositories := getAllSections(section, "packages/pacman/repository")
	for _, repo := range repositories {
		repoName := repo.GetFirst("name", "")
		if repoName != "" {
			repoModifications := map[string]interface{}{
				"Include": repo.GetFirst("include", ""),
				"Server":  repo.GetFirst("server", ""),
			}
			if repoModifications["Include"] == "" && repoModifications["Server"] == "" && slices.Contains(builtinRepositories, repoName) {
				repoModifications["Include"] = "/etc/pacman.d/mirrorlist"
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

// This function is the same as getAll() but automatically adds the `+bare` tag to the last item.
func getAllKernels(section *parser.Section, key string) []string {
	kernels := getAll(section, key)
	items := section.GetAll(key)
	if len(items) > 0 && (len(kernels) == 0 || kernels[len(kernels)-1] != items[len(items)-1]) {
		kernels = append(kernels, items[len(items)-1])
	}
	return kernels
}

// This function is the same as section.GetAll(), but if an item has spaces, it will be split into multiple items.
// It also supports tags. For example, `package = linux-headers linux-firmware, +bare` needs to be split into `linux-headers` and `linux-firmware`.
func getAll(section *parser.Section, key string) []string {
	items := section.GetAll(key)
	result := []string{}
	for i := 0; i < len(items); i++ {
		included := true

		parts := strings.Split(items[i], ",")

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
	applyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file")
	applyCmd.PersistentFlags().BoolP("bare", "b", false, "Install only essential packages with the +bare tag (equivalent to --tags=\"-default +bare\")")

	applyCmd.PersistentFlags().StringSliceVar(&tags, "tags", []string{}, "List of tags to include/exclude, e.g. '-default +bare'")

	applyCmd.PersistentFlags().BoolP("upgrade", "u", false, "Perform a system upgrade")

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
