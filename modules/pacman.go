package modules

import (
	"strings"

	"github.com/DevReaper0/declarch/utils"
)

func PacmanInstall(pkg string) error {
	splitPkg := strings.Split(pkg, " ")
	err := utils.ExecCommand(append([]string{
		"pacman", "-S", "--needed", "--noconfirm",
	}, splitPkg...), "", true)
	if err != nil {
		return err
	}
	return nil
}

func PacmanRemove(pkg string) error {
	splitPkg := strings.Split(pkg, " ")
	err := utils.ExecCommand(append([]string{
		"pacman", "-R", "--noconfirm",
	}, splitPkg...), "", true)
	if err != nil {
		return err
	}
	return nil
}
