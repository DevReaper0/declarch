package modules

import (
	"fmt"

	"github.com/DevReaper0/declarch/utils"
)

func PacmanInstall(pkgs interface{}) error {
	pkgNames, ok := pkgs.([]string)
	if !ok {
		return fmt.Errorf("expected []string for package names, got %T", pkgs)
	}
	return utils.ExecCommand(append([]string{
		"pacman", "-S", "--needed", "--noconfirm",
	}, pkgNames...), "", "")
}

func PacmanRemove(pkgs interface{}) error {
	pkgNames, ok := pkgs.([]string)
	if !ok {
		return fmt.Errorf("expected []string for package names, got %T", pkgs)
	}

	return utils.ExecCommand(append([]string{
		"pacman", "-R", "--noconfirm",
	}, pkgNames...), "", "")
}

func PacmanSystemUpgrade() error {
	return utils.ExecCommand([]string{
		"pacman", "-Syu", "--noconfirm",
	}, "", "")
}