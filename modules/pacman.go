package modules

import (
	"strings"

	"github.com/DevReaper0/declarch/utils"
)

func PacmanInstall(pkg string) error {
	splitPkg := strings.Split(pkg, " ")
	err := utils.ExecCommand(append([]string{
		"pacman", "-S", "--needed", "--noconfirm",
	}, splitPkg...), "", "")
	if err != nil {
		return err
	}
	return nil
}

func PacmanRemove(pkg string) error {
	splitPkg := strings.Split(pkg, " ")
	err := utils.ExecCommand(append([]string{
		"pacman", "-R", "--noconfirm",
	}, splitPkg...), "", "")
	if err != nil {
		return err
	}
	return nil
}

func PacmanSystemUpgrade() error {
	err := utils.ExecCommand([]string{
		"pacman", "-Syu", "--noconfirm",
	}, "", "")
	if err != nil {
		return err
	}
	return nil
}
