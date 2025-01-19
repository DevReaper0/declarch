package modules

import (
	"slices"
	"strings"

	"github.com/DevReaper0/declarch/utils"
)

func AURInstall(helper string, pkg string) error {
	if helper == "makepkg" {
		return MakepkgInstall(pkg)
	}
	return PacmanWrapperInstall(helper, pkg)
}

func PacmanWrapperInstall(wrapper string, pkg string) error {
	splitPkg := strings.Split(pkg, " ")
	err := utils.ExecCommand(append([]string{
		wrapper, "-S", "--needed", "--noconfirm",
	}, splitPkg...), "", slices.Contains(rootPacmanWrappers, wrapper))
	if err != nil {
		return err
	}
	return nil
}

var rootPacmanWrappers = []string{}
