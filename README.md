# DeclArch

This is a work-in-progress tool for declaratively managing an Arch Linux system.

Currently, this repository contains only a parser and the framework for the CLI.

To try out the parser, ensure the Go programming language is installed, clone this repository, and run:

```sh
go build .
./declarch apply -c test.conf
```

You can find the test code in `cmds/apply.go` and the test data in `test.conf` and `testb.conf` (imported by `test.conf`).

If you don't want to run it yourself, the test's output would be:
```
2
[2]

2
[2]

3
[3]

3, green, blue
[3, green, blue red, green, blue]

2, green, blue
[2, green, blue]

3, 26, Warsaw
[3, 26, Warsaw Andrew, 21, Berlin Koichi, 18, Morioh]
```

There is also an example DeclArch configuration in `config.conf`.
