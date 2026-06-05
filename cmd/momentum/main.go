package main

import (
	"fmt"
	"os"

	"github.com/theinventor/momentumticker-cli/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
