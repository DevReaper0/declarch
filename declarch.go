package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/user"

	"github.com/DevReaper0/declarch/cmds"
	"github.com/fatih/color"
)

//go:embed default_declarch.conf
var defaultConfig string

func main() {
	currentUser, err := user.Current()
	if err != nil {
		color.Set(color.FgRed)
		fmt.Print("Error getting current user: ")
		color.Unset()
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if currentUser.Uid != "0" {
		color.Set(color.FgRed)
		fmt.Println("User must be root.")
		fmt.Println("Please rerun the command as root.")
		color.Unset()
		return
	}

	cmds.Execute(defaultConfig)
}