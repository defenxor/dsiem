//go:generate go-extpoints
package main

import (
	"dsiem/pkg/intel"
	"fmt"
	"os"
)

var intelPlugins = intel.Checkers

func usage() {
	fmt.Println("Available intel plugins:\n")
	for name := range intelPlugins.All() {
		fmt.Println(" - ", name)
	}
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := intelPlugins.Lookup(os.Args[1])
	if cmd == nil {
		usage()
	}
	cmd.CheckIP("10.7.2.1")
}
