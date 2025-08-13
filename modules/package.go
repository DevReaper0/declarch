package modules

import (
	"github.com/DevReaper0/declarch/parser"
)

type Package struct {
	value interface{}
	hooks []Hook
}

func NewPackage(value interface{}) *Package {
	return &Package{
		value: value,
		hooks: make([]Hook, 0),
	}
}

func (p *Package) AddHook(section *parser.Section) error {
	hook, err := HookFrom(section, "install", "remove")
	if err != nil {
		return err
	}

	p.hooks = append(p.hooks, hook)
	return nil
}

type PackageList struct {
	Packages    []*Package
	InstallFunc func(interface{}) error
	RemoveFunc  func(interface{}) error
}

func NewPackageList(installFunc, removeFunc func(interface{}) error) *PackageList {
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
		for _, hook := range pkg.hooks {
			if hook.For == "install" && hook.When == "before" {
				if err := hook.Exec(); err != nil {
					return err
				}
			}
		}
	}

	// Install all packages in one command
	if len(pl.Packages) > 0 {
		values := make([]interface{}, len(pl.Packages))
		for i, pkg := range pl.Packages {
			values[i] = pkg.value
		}

		if err := pl.InstallFunc(values); err != nil {
			return err
		}
	}

	// Run after hooks for all packages
	for _, pkg := range pl.Packages {
		for _, hook := range pkg.hooks {
			if hook.For == "install" && hook.When == "after" {
				if err := hook.Exec(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (pl *PackageList) Remove() error {
	// Run before hooks for all packages
	for _, pkg := range pl.Packages {
		for _, hook := range pkg.hooks {
			if hook.For == "remove" && hook.When == "before" {
				if err := hook.Exec(); err != nil {
					return err
				}
			}
		}
	}

	// Remove all packages in one command
	if len(pl.Packages) > 0 {
		values := make([]interface{}, len(pl.Packages))
		for i, pkg := range pl.Packages {
			values[i] = pkg.value
		}

		if err := pl.RemoveFunc(values); err != nil {
			return err
		}
	}

	// Run after hooks for all packages
	for _, pkg := range pl.Packages {
		for _, hook := range pkg.hooks {
			if hook.For == "remove" && hook.When == "after" {
				if err := hook.Exec(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}