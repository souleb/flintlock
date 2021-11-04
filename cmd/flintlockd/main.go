package main

import (
	"fmt"
	"os"

	"github.com/weaveworks/flintlock/internal/command"
)

func main() {
	app := command.NewApp()

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "flintlockd: %s\n", err)
		os.Exit(1)
	}
}
