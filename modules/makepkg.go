package modules

import (
	"os"

	"github.com/DevReaper0/declarch/utils"
)

func MakepkgInstall(pkgName string) error {
	dir, err := os.MkdirTemp("", pkgName)
	if err != nil {
		return err
	}

	err = utils.Chown(dir, utils.NormalUser)
	if err != nil {
		return err
	}

	err = utils.ExecCommand([]string{
		"git", "clone", "https://aur.archlinux.org/" + pkgName + ".git", dir,
	}, "", utils.NormalUser)
	if err != nil {
		return err
	}

	err = utils.ExecCommand([]string{
		"makepkg", "-si", "--needed", "--noconfirm",
	}, dir, utils.NormalUser)
	if err != nil {
		return err
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return err
	}

	return nil
}