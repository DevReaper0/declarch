package cmds

import (
	"fmt"
	"os"
	"os/user"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var defaultConfig string

var rootCmd = &cobra.Command{
	Use:   "declarch",
	Short: "DeclArch is a tool for declaratively managing an Arch Linux system",
	Long:  "DeclArch is a tool for declaratively managing an Arch Linux system",
}

func Execute(defaultCfg string) {
	defaultConfig = defaultCfg
	rootCmd.Execute()
}

// CheckRoot checks if the current user is root and prints an error message if not
// Returns true if the user is root, false otherwise
func CheckRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		color.Set(color.FgRed)
		fmt.Print("Error getting current user: ")
		color.Unset()
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	if currentUser.Uid != "0" {
		color.Set(color.FgRed)
		fmt.Println("This command requires root privileges.")
		fmt.Println("Please rerun the command as root.")
		color.Unset()
		return false
	}

	return true
}