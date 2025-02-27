package modules

import (
	"strings"

	"github.com/DevReaper0/declarch/utils"
)

type PackageHook struct {
	Timing string // "before" or "after"
	User   string
	Run    string
}

type Package struct {
	Name  string
	Hooks []PackageHook
}

func NewPackage(name string) *Package {
	return &Package{
		Name:  name,
		Hooks: make([]PackageHook, 0),
	}
}

func (p *Package) AddHook(timing, user, run string) {
	p.Hooks = append(p.Hooks, PackageHook{
		Timing: timing,
		User:   user,
		Run:    run,
	})
}

type PackageList struct {
	Packages    []*Package
	InstallFunc func(string) error
	RemoveFunc  func(string) error
}

func NewPackageList(installFunc, removeFunc func(string) error) *PackageList {
	return &PackageList{
		Packages:    make([]*Package, 0),
		InstallFunc: installFunc,
		RemoveFunc:  removeFunc,
	}
}

func (pl *PackageList) Add(pkg *Package) {
	pl.Packages = append(pl.Packages, pkg)
}

func (pl *PackageList) Clear() {
	pl.Packages = make([]*Package, 0)
}

func (pl *PackageList) Install() error {
	// Run before hooks for all packages
	for _, pkg := range pl.Packages {
		for _, hook := range pkg.Hooks {
			if hook.Timing == "before" && hook.Run != "" {
				if err := runHook(hook); err != nil {
					return err
				}
			}
		}
	}

	// Install all packages in one command
	if len(pl.Packages) > 0 {
		names := make([]string, 0, len(pl.Packages))
		for _, pkg := range pl.Packages {
			names = append(names, pkg.Name)
		}
		if err := pl.InstallFunc(strings.Join(names, " ")); err != nil {
			return err
		}
	}

	// Run after hooks for all packages
	for _, pkg := range pl.Packages {
		for _, hook := range pkg.Hooks {
			if hook.Timing == "after" && hook.Run != "" {
				if err := runHook(hook); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (pl *PackageList) Remove() error {
	if len(pl.Packages) > 0 {
		names := make([]string, 0, len(pl.Packages))
		for _, pkg := range pl.Packages {
			names = append(names, pkg.Name)
		}
		if err := pl.RemoveFunc(strings.Join(names, " ")); err != nil {
			return err
		}
	}
	return nil
}

func runHook(hook PackageHook) error {
	return utils.ExecCommand(strings.Fields(hook.Run), "", hook.User)
}
