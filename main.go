package main

import "github.com/micahlee/pco-cli/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
