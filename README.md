# DeclArch

This is a work-in-progress tool for declaratively managing an Arch Linux system.

Currently, this repository contains only a parser and the framework for the CLI.

To try out the parser, ensure the Go programming language is installed, clone this repository, and run:

```sh
go build .
./declarch apply -c test.conf
```

You can find the test code in `cmds/apply.go` and the test data in `test.conf` and `testb.conf` (imported by `test.conf`).

There is also an example DeclArch configuration in `config.conf`.
