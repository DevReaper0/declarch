package modules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DevReaper0/declarch/parser"
	"github.com/DevReaper0/declarch/utils"
)

type FlatpakPackage struct {
	Name             string
	Remote           string
	UserInstallation bool
	Installation     string
	Architecture     string
	Subpath          string
}

func FlatpakPackageFrom(input interface{}) (FlatpakPackage, error) {
	pkg := FlatpakPackage{}

	switch v := input.(type) {
	case *parser.Section:
		if name := v.GetFirst("name", ""); name != "" {
			pkg.Name = name
		} else {
			return pkg, fmt.Errorf("Flatpak package section is missing 'name' field")
		}

		pkg.Remote = v.GetFirst("remote", "")

		{
			userInstallationString := v.GetFirst("user_installation", "false")
			userInstallation, err := strconv.ParseBool(userInstallationString)
			if err != nil {
				return pkg, fmt.Errorf("invalid value for 'user_installation' field in Flatpak package section '%s': %s", pkg.Name, userInstallationString)
			}
			pkg.UserInstallation = userInstallation
		}
		pkg.Installation = v.GetFirst("installation", "")

		pkg.Architecture = v.GetFirst("architecture", "")
		pkg.Subpath = v.GetFirst("subpath", "")
	case string:
		pkg.Name = v
	default:
		return pkg, fmt.Errorf("unsupported type for FlatpakPackage: %T", input)
	}
	return pkg, nil
}

type FlatpakRemote struct {
	Name             string
	URL              string
	UserInstallation bool
	Installation     string
	Disable          bool
	Title            string
	Comment          string
	Description      string
	Homepage         string
	Icon             string
	DefaultBranch    string
}

func FlatpakRemoteFrom(input interface{}) (FlatpakRemote, error) {
	remote := FlatpakRemote{}

	switch v := input.(type) {
	case *parser.Section:
		if name := v.GetFirst("name", ""); name != "" {
			remote.Name = name
		} else {
			return remote, fmt.Errorf("Flatpak remote section is missing 'name' field")
		}

		if url := v.GetFirst("url", ""); url != "" {
			remote.URL = url
		} else {
			return remote, fmt.Errorf("Flatpak remote section '%s' is missing 'url' field", remote.Name)
		}

		{
			userInstallationString := v.GetFirst("user_installation", "false")
			userInstallation, err := strconv.ParseBool(userInstallationString)
			if err != nil {
				return remote, fmt.Errorf("invalid value for 'user_installation' field in Flatpak remote section '%s': %s", remote.Name, userInstallationString)
			}
			remote.UserInstallation = userInstallation
		}
		remote.Installation = v.GetFirst("installation", "")

		{
			disableString := v.GetFirst("disable", "false")
			disable, err := strconv.ParseBool(disableString)
			if err != nil {
				return remote, fmt.Errorf("invalid value for 'disable' field in Flatpak remote section '%s': %s", remote.Name, disableString)
			}
			remote.Disable = disable
		}

		remote.Title = v.GetFirst("title", "")
		remote.Comment = v.GetFirst("comment", "")
		remote.Description = v.GetFirst("description", "")
		remote.Homepage = v.GetFirst("homepage", "")
		remote.Icon = v.GetFirst("icon", "")
		remote.DefaultBranch = v.GetFirst("default_branch", "")
	case string:
		remote.Name = v
	default:
		return remote, fmt.Errorf("unsupported type for FlatpakRemote: %T", input)
	}
	return remote, nil
}

func FlatpakInstall(pkgs interface{}) error {
	pkgObjs, ok := pkgs.([]FlatpakPackage)
	if !ok {
		return fmt.Errorf("expected []FlatpakPackage for package objects, got %T", pkgs)
	}

	for _, pkgObj := range pkgObjs {
		args := []string{"flatpak", "install", "--noninteractive", "--assumeyes"}

		if pkgObj.UserInstallation {
			args = append(args, "--user")
		}

		if pkgObj.Installation != "" {
			args = append(args, "--installation="+pkgObj.Installation)
		}

		if pkgObj.Architecture != "" {
			args = append(args, "--arch="+pkgObj.Architecture)
		}

		if pkgObj.Subpath != "" {
			args = append(args, "--subpath="+pkgObj.Subpath)
		}

		if pkgObj.Remote != "" {
			args = append(args, pkgObj.Remote)
		}

		splitPkg := strings.Fields(pkgObj.Name)
		args = append(args, splitPkg...)

		if err := utils.ExecCommand(args, "", PrimaryUser); err != nil {
			return fmt.Errorf("Failed to install flatpak package '%s': %w", pkgObj.Name, err)
		}
	}

	return nil
}

func FlatpakRemove(pkgs interface{}) error {
	pkgObjs, ok := pkgs.([]FlatpakPackage)
	if !ok {
		return fmt.Errorf("expected []FlatpakPackage for package objects, got %T", pkgs)
	}

	for _, pkgObj := range pkgObjs {
		args := []string{"flatpak", "uninstall", "--noninteractive", "--assumeyes"}

		if pkgObj.UserInstallation {
			args = append(args, "--user")
		}

		if pkgObj.Installation != "" {
			args = append(args, "--installation="+pkgObj.Installation)
		}

		if pkgObj.Architecture != "" {
			args = append(args, "--arch="+pkgObj.Architecture)
		}

		splitPkg := strings.Fields(pkgObj.Name)
		args = append(args, splitPkg...)

		if err := utils.ExecCommand(args, "", PrimaryUser); err != nil {
			return fmt.Errorf("Failed to remove flatpak package '%s': %w", pkgObj.Name, err)
		}
	}

	return nil
}

func FlatpakSystemUpgrade() error {
	return utils.ExecCommand([]string{
		"flatpak", "update", "--noninteractive", "--assumeyes",
	}, "", PrimaryUser)
}

func FlatpakAddRemote(remote FlatpakRemote) error {
	args := []string{"flatpak", "remote-add", "--if-not-exists"}

	if remote.UserInstallation {
		args = append(args, "--user")
	}

	if remote.Installation != "" {
		args = append(args, "--installation="+remote.Installation)
	}

	if remote.Disable {
		args = append(args, "--disable")
	}

	if remote.Title != "" {
		args = append(args, "--title="+remote.Title)
	}

	if remote.Comment != "" {
		args = append(args, "--comment="+remote.Comment)
	}

	if remote.Description != "" {
		args = append(args, "--description="+remote.Description)
	}

	if remote.Homepage != "" {
		args = append(args, "--homepage="+remote.Homepage)
	}

	if remote.Icon != "" {
		args = append(args, "--icon="+remote.Icon)
	}

	if remote.DefaultBranch != "" {
		args = append(args, "--default-branch="+remote.DefaultBranch)
	}

	args = append(args, remote.Name, remote.URL)

	return utils.ExecCommand(args, "", PrimaryUser)
}

func FlatpakRemoveRemote(remote FlatpakRemote) error {
	args := []string{"flatpak", "remote-delete"}

	if remote.UserInstallation {
		args = append(args, "--user")
	}

	if remote.Installation != "" {
		args = append(args, "--installation="+remote.Installation)
	}

	args = append(args, remote.Name)

	return utils.ExecCommand(args, "", PrimaryUser)
}

func FlatpakModifyRemote(remote FlatpakRemote) error {
	args := []string{"flatpak", "remote-modify"}

	if remote.UserInstallation {
		args = append(args, "--user")
	}

	if remote.Installation != "" {
		args = append(args, "--installation="+remote.Installation)
	}

	if remote.Disable {
		args = append(args, "--disable")
	} else {
		args = append(args, "--enable")
	}

	if remote.Title != "" {
		args = append(args, "--title="+remote.Title)
	}

	if remote.Comment != "" {
		args = append(args, "--comment="+remote.Comment)
	}

	if remote.Description != "" {
		args = append(args, "--description="+remote.Description)
	}

	if remote.Homepage != "" {
		args = append(args, "--homepage="+remote.Homepage)
	}

	if remote.Icon != "" {
		args = append(args, "--icon="+remote.Icon)
	}

	if remote.DefaultBranch != "" {
		args = append(args, "--default-branch="+remote.DefaultBranch)
	}

	args = append(args, remote.Name)

	return utils.ExecCommand(args, "", PrimaryUser)
}