package main

import (
	"fmt"
	"os"

	"github.com/fatih/starhook/internal/command"
)

func main() {
	if err := command.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
