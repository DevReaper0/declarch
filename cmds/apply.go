package cmds

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/DevReaper0/declarch/modules"
	"github.com/DevReaper0/declarch/parser"
	"github.com/DevReaper0/declarch/utils"
)

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
	//

	modules.PrivilegeEscalationCommand = section.GetFirst("essentials/privilige_escalation", "sudo")

	// TODO: Pacman configuration:
	// TODO: color
	// TODO: parallel_downloads
	// TODO: repositories

	if err := modules.PacmanInstall("base base-devel git"); err != nil {
		return err
	}

	// Temporary fix the non-root commands since user management is not implemented yet.
	// For now, a user must be defined the configuration and the user must already exist on the system.
	// Otherwise, the "nobody" user will be used.
	utils.NormalUser = section.GetFirst("users/user/username", "nobody")

	addedKernels, removedKernels := utils.GetDifferences(section.GetAll("essentials/kernel"), previousSection.GetAll("essentials/kernel"))
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

	addedPacmanPackages, removedPacmanPackages := utils.GetDifferences(section.GetAll("packages/pacman/package"), previousSection.GetAll("packages/pacman/package"))
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

	addedAurPackages, removedAurPackages := utils.GetDifferences(section.GetAll("packages/aur/package"), previousSection.GetAll("packages/aur/package"))
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

func init() {
	applyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/declarch.conf", "Configuration file (default is /etc/declarch/declarch.conf)")

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
