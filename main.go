package main

import (
	"os"

	"github.com/Bader-GmbH/iot-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
