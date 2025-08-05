package main

import (
	_ "embed"

	"github.com/DevReaper0/declarch/cmds"
)

//go:embed default_declarch.conf
var defaultConfig string

func main() {
	cmds.Execute(defaultConfig)
}