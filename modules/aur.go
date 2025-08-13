package modules

import (
	"fmt"
	"slices"

	"github.com/DevReaper0/declarch/utils"
)

func AURInstall(helper string, pkgs interface{}) error {
	pkgNames, ok := pkgs.([]string)
	if !ok {
		return fmt.Errorf("expected []string for package names, got %T", pkgs)
	}

	if helper == "makepkg" {
		for _, pkgName := range pkgNames {
			if err := MakepkgInstall(pkgName); err != nil {
				return err
			}
		}
		return nil
	}
	return PacmanWrapperInstall(helper, pkgNames)
}

func PacmanWrapperInstall(wrapper string, pkgNames []string) error {
	user := ""
	if slices.Contains(rootPacmanWrappers, wrapper) {
		user = ""
	} else {
		user = PrimaryUser
	}

	return utils.ExecCommand(append([]string{
		wrapper, "-S", "--needed", "--noconfirm",
	}, pkgNames...), "", user)
}

func PacmanWrapperSystemUpgrade(wrapper string) error {
	user := ""
	if slices.Contains(rootPacmanWrappers, wrapper) {
		user = ""
	} else {
		user = PrimaryUser
	}

	return utils.ExecCommand([]string{
		wrapper, "-Syu", "--noconfirm",
	}, "", user)
}

var rootPacmanWrappers = []string{}