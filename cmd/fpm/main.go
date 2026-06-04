package main

import (
	"os"

	"github.com/kartikeyyadav/fpm/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
