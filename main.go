package main

import (
	"os"

	"github.com/DasDarki/nwm/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
