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
var tagSet *modules.TagSet

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if !CheckRoot() {
			return
		}

		configPath, _ := cmd.Flags().GetString("config")
		configPath, _ = filepath.Abs(configPath)

		tagSet = modules.NewTagSet("+default")

		if tags, _ := cmd.PersistentFlags().GetStringSlice("tags"); len(tags) > 0 {
			tagSet.AddTags(tags)
		}

		if bare, _ := cmd.PersistentFlags().GetBool("bare"); bare {
			tagSet.AddTags([]string{"-default", "+bare"})
		}

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

	if err := configurePacman(section); err != nil {
		return fmt.Errorf("error configuring pacman: %w", err)
	}

	if err := modules.PacmanInstall([]string{"base", "base-devel", "git"}); err != nil {
		return err
	}

	// Temporary fix for the non-root commands since user management is not implemented yet.
	// For now, a user must be defined the configuration and the user must already exist on the system.
	// Otherwise, the "nobody" user will be used.
	utils.NormalUser = section.GetFirst("users/user/username", "nobody")

	if err := applyKernels(section, previousSection); err != nil {
		return fmt.Errorf("error applying kernel configuration: %w", err)
	}

	if err := applyBootloader(section, previousSection); err != nil {
		return fmt.Errorf("error applying bootloader configuration: %w", err)
	}

	if err := applyPacman(section, previousSection); err != nil {
		return fmt.Errorf("error applying pacman configuration: %w", err)
	}

	if err := applyAUR(section, previousSection); err != nil {
		return fmt.Errorf("error applying AUR configuration: %w", err)
	}

	if err := applyFlatpak(section, previousSection); err != nil {
		return fmt.Errorf("error applying Flatpak configuration: %w", err)
	}

	if err := applyNetworkHandler(section, previousSection); err != nil {
		return fmt.Errorf("error applying network handler configuration: %w", err)
	}

	return nil
}

func applyKernels(section *parser.Section, previousSection *parser.Section) error {
	packageCommandHooks := getAllSections(section, "packages/pacman/hook")

	kernelList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedKernels, removedKernels := utils.GetDifferences(tagSet.GetAll(section, "essentials/kernel"), tagSet.GetAll(previousSection, "essentials/kernel"))

	// Installing a kernel before removing any just to be safe
	if len(addedKernels) > 0 {
		kernelPackageNames := section.GetAll("essentials/kernel")
		pkgNamesString := kernelPackageNames[len(kernelPackageNames)-1]
		pkgNamesString = strings.SplitN(pkgNamesString, ",", 2)[0]
		pkgNames := strings.Fields(pkgNamesString)
		for _, pkgName := range pkgNames {
			pkg := modules.NewPackage(pkgName)
			for _, hook := range packageCommandHooks {
				if hook.GetFirst("package", "") == pkgName && hook.GetFirst("for", "install") == "install" {
					pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
				}
			}
			kernelList.Add(pkg)
		}

		addedKernels = slices.DeleteFunc(addedKernels, func(s string) bool {
			return slices.Contains(pkgNames, s)
		})

		if err := kernelList.Install(); err != nil {
			return err
		}
		kernelList.Clear()
	}

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

	for _, pkgName := range addedKernels {
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

	return nil
}

func applyBootloader(section *parser.Section, previousSection *parser.Section) error {
	packageCommandHooks := getAllSections(section, "packages/pacman/hook")

	bootloaderList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedBootloader, removedBootloader := utils.GetDifferences(strings.Fields(section.GetFirst("essentials/bootloader", "grub efibootmgr")), strings.Fields(previousSection.GetFirst("essentials/bootloader", "")))

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
	if err := bootloaderList.Install(); err != nil {
		return err
	}

	return nil
}

func applyNetworkHandler(section *parser.Section, previousSection *parser.Section) error {
	packageCommandHooks := getAllSections(section, "packages/pacman/hook")

	networkHandlerList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedNetworkHandler, removedNetworkHandler := utils.GetDifferences(strings.Fields(section.GetFirst("essentials/network_handler", "networkmanager")), strings.Fields(previousSection.GetFirst("essentials/network_handler", "")))

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

	return nil
}

func applyPacman(section *parser.Section, previousSection *parser.Section) error {
	packageCommandHooks := getAllSections(section, "packages/pacman/hook")

	pacmanList := modules.NewPackageList(modules.PacmanInstall, modules.PacmanRemove)
	addedPacmanPackages, removedPacmanPackages := utils.GetDifferences(tagSet.GetAll(section, "packages/pacman/package"), tagSet.GetAll(previousSection, "packages/pacman/package"))

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

	return nil
}

func applyAUR(section *parser.Section, previousSection *parser.Section) error {
	packageCommandHooks := getAllSections(section, "packages/aur/hook")

	aurHelper := section.GetFirst("packages/aur/helper", "makepkg")
	aurInstall := func(pkgs interface{}) error { return modules.AURInstall(aurHelper, pkgs) }
	aurList := modules.NewPackageList(aurInstall, modules.PacmanRemove)
	addedAurPackages, removedAurPackages := utils.GetDifferences(tagSet.GetAll(section, "packages/aur/package"), tagSet.GetAll(previousSection, "packages/aur/package"))

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

	return nil
}

func applyFlatpak(section *parser.Section, previousSection *parser.Section) error {
	packageCommandHooks := getAllSections(section, "packages/flatpak/hook")

	autoInstallString := section.GetFirst("packages/flatpak/auto_install", "true")
	autoInstall, err := strconv.ParseBool(autoInstallString)
	if err != nil {
		return fmt.Errorf("invalid value for 'auto_install' field in Flatpak section: %s", autoInstallString)
	}
	if autoInstall {
		flatpakPackages := getFlatpakPackages(section)
		if len(flatpakPackages) > 0 {
			if err := modules.PacmanInstall("flatpak"); err != nil {
				return err
			}
		}
	}

	currentRemotes := getAllSections(section, "packages/flatpak/remote")
	previousRemotes := getAllSections(previousSection, "packages/flatpak/remote")

	addedRemotes, removedRemotes := utils.GetDifferences(getFlatpakRemoteIdentifiers(currentRemotes), getFlatpakRemoteIdentifiers(previousRemotes))

	previousRemoteMap := make(map[string]modules.FlatpakRemote)
	for _, remote := range previousRemotes {
		name := remote.GetFirst("name", "")
		if name == "" {
			continue
		}

		remoteObj, err := modules.FlatpakRemoteFrom(remote)
		if err != nil {
			return fmt.Errorf("error processing previous remote %s: %w", name, err)
		}

		identifier := createFlatpakIdentifier(remoteObj.Name, remoteObj.Installation, remoteObj.UserInstallation)
		previousRemoteMap[identifier] = remoteObj
	}

	// Remove remotes that are no longer configured
	for _, identifier := range removedRemotes {
		prevRemote := previousRemoteMap[identifier]
		if err := modules.FlatpakRemoveRemote(prevRemote); err != nil {
			return err
		}
	}

	for _, remote := range currentRemotes {
		remoteObj, err := modules.FlatpakRemoteFrom(remote)
		if err != nil {
			return err
		}

		identifier := createFlatpakIdentifier(remoteObj.Name, remoteObj.Installation, remoteObj.UserInstallation)

		if slices.Contains(addedRemotes, identifier) {
			if err := modules.FlatpakAddRemote(remoteObj); err != nil {
				return err
			}
		} else if remoteObj != previousRemoteMap[identifier] {
			if err := modules.FlatpakModifyRemote(remoteObj); err != nil {
				return fmt.Errorf("error updating remote %s: %w", remoteObj.Name, err)
			}
		}
	}

	flatpakList := modules.NewPackageList(modules.FlatpakInstall, modules.FlatpakRemove)
	currentFlatpakPackages := getFlatpakPackages(section)
	previousFlatpakPackages := getFlatpakPackages(previousSection)
	addedFlatpakPackages, removedFlatpakPackages := utils.GetDifferences(
		getFlatpakPackageIdentifiers(currentFlatpakPackages),
		getFlatpakPackageIdentifiers(previousFlatpakPackages),
	)

	addedPackageObjs := make(map[string]modules.FlatpakPackage)
	for _, pkgIdentifier := range addedFlatpakPackages {
		for _, pkg := range currentFlatpakPackages {
			identifier := createFlatpakIdentifier(pkg.Name, pkg.Installation, pkg.UserInstallation)
			if identifier == pkgIdentifier {
				addedPackageObjs[pkgIdentifier] = pkg
				break
			}
		}
	}

	removedPackageObjs := make(map[string]modules.FlatpakPackage)
	for _, pkgIdentifier := range removedFlatpakPackages {
		for _, pkg := range previousFlatpakPackages {
			identifier := createFlatpakIdentifier(pkg.Name, pkg.Installation, pkg.UserInstallation)
			if identifier == pkgIdentifier {
				removedPackageObjs[pkgIdentifier] = pkg
				break
			}
		}
	}

	for _, pkgIdentifier := range removedFlatpakPackages {
		flatpakPackage := removedPackageObjs[pkgIdentifier]
		pkg := modules.NewPackage(flatpakPackage)

		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgIdentifier && hook.GetFirst("for", "install") == "remove" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		flatpakList.Add(pkg)
	}
	if err := flatpakList.Remove(); err != nil {
		return err
	}
	flatpakList.Clear()

	for _, pkgIdentifier := range addedFlatpakPackages {
		flatpakPackage := addedPackageObjs[pkgIdentifier]
		pkg := modules.NewPackage(flatpakPackage)

		for _, hook := range packageCommandHooks {
			if hook.GetFirst("package", "") == pkgIdentifier && hook.GetFirst("for", "install") == "install" {
				pkg.AddHook(hook.GetFirst("timing", "after"), hook.GetFirst("user", ""), hook.GetFirst("run", ""))
			}
		}
		flatpakList.Add(pkg)
	}
	if err := flatpakList.Install(); err != nil {
		return err
	}

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

	pacmanPackageCount := len(section.GetAll("packages/pacman/package"))
	pacmanConfigured := pacmanPackageCount > 0

	aurHelper := section.GetFirst("packages/aur/helper", "")
	aurPackageCount := len(section.GetAll("packages/aur/package"))
	aurConfigured := aurHelper != "" && aurPackageCount > 0

	flatpakPackageCount := len(section.GetAll("packages/flatpak/package")) + len(section.GetAll("packages/flatpak/package/name"))
	flatpakConfigured := flatpakPackageCount > 0

	availableUpgrades := []string{}
	if pacmanConfigured {
		availableUpgrades = append(availableUpgrades, "pacman:system packages via Pacman")
	}
	if aurConfigured {
		availableUpgrades = append(availableUpgrades, "aur:AUR packages via "+aurHelper)
	}
	if flatpakConfigured {
		availableUpgrades = append(availableUpgrades, "flatpak:Flatpak packages")
	}

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

	if slices.Contains(toUpgrade, "flatpak") {
		color.Set(color.FgCyan)
		fmt.Println("Upgrading Flatpak packages...")
		color.Unset()

		if err := modules.FlatpakSystemUpgrade(); err != nil {
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
		color.Set(color.FgRed)
		fmt.Println("Error reading input:", err)
		color.Unset()
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
			color.Set(color.FgRed)
			fmt.Println("Error reading input:", err)
			color.Unset()
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

	replaceCommentsString := section.GetFirst("config_parser/replace_comments", "true")
	replaceComments, err := strconv.ParseBool(replaceCommentsString)
	if err != nil {
		return fmt.Errorf("invalid value for 'replace_comments' field in config_parser section: %s", replaceCommentsString)
	}
	pacmanPatcher.ReplaceComments = replaceComments

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

// createFlatpakIdentifier creates a standardized identifier for Flatpak resources
// Format: "[installation]:name" if specific system-wide installation was specified
// Format: ":name" if specified to use per-user installation
// Format: "default:name" if no installation was specified (default system-wide installation)
func createFlatpakIdentifier(name, installation string, userInstallation bool) string {
	if installation != "" {
		return installation + ":" + name
	}
	if userInstallation {
		return ":" + name
	}
	return "default:" + name
}

func getFlatpakRemoteIdentifiers(remotes []*parser.Section) []string {
	identifiers := []string{}
	for _, remote := range remotes {
		name := remote.GetFirst("name", "")

		userInstallString := remote.GetFirst("user_installation", "false")
		userInstall, err := strconv.ParseBool(userInstallString)
		if err != nil {
			color.Set(color.FgRed)
			fmt.Printf("Error parsing 'user_installation' field for remote '%s': %v\n", name, err)
			color.Unset()
			continue
		}

		installation := remote.GetFirst("installation", "")

		if name != "" {
			identifiers = append(identifiers, createFlatpakIdentifier(name, installation, userInstall))
		}
	}
	return identifiers
}

func getFlatpakPackages(section *parser.Section) []modules.FlatpakPackage {
	if section == nil {
		return []modules.FlatpakPackage{}
	}

	packages := []modules.FlatpakPackage{}

	// Process packages from values
	pkgNames := tagSet.GetAll(section, "packages/flatpak/package")
	for _, pkgName := range pkgNames {
		pkg, err := modules.FlatpakPackageFrom(pkgName)
		if err != nil {
			color.Set(color.FgRed)
			fmt.Printf("Error parsing Flatpak package value: %v\n", err)
			color.Unset()
			continue
		}
		packages = append(packages, pkg)
	}

	// Process packages from sections
	sections := getAllSections(section, "packages/flatpak/package")
	for _, pkgSection := range sections {
		names := tagSet.GetAll(pkgSection, "name")
		for _, name := range names {
			pkg, err := modules.FlatpakPackageFrom(pkgSection)
			if err != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error parsing Flatpak package section: %v\n", err)
				color.Unset()
				continue
			}
			pkg.Name = name
			packages = append(packages, pkg)
		}
	}

	return packages
}

func getFlatpakPackageIdentifiers(packages []modules.FlatpakPackage) []string {
	names := make([]string, len(packages))
	for i, pkg := range packages {
		names[i] = createFlatpakIdentifier(pkg.Name, pkg.Installation, pkg.UserInstallation)
	}
	return names
}

func init() {
	applyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file")
	applyCmd.PersistentFlags().BoolP("bare", "b", false, "Install only essential packages with the +bare tag (equivalent to --tags=\"-default +bare\")")

	applyCmd.PersistentFlags().StringSlice("tags", []string{}, "List of tags to include/exclude, e.g. '-default +bare'")

	applyCmd.PersistentFlags().BoolP("upgrade", "u", false, "Perform a system upgrade")

	rootCmd.AddCommand(applyCmd)
}