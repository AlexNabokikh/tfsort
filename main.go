package main

import (
	"github.com/AlexNabokikh/tfsort/cmd"
)

// Build time variables are set using -ldflags.
var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals // Set by ldflags at build time
	date    = "unknown" //nolint:gochecknoglobals // Set by ldflags at build time
)

func main() {
	cmd.Execute(version, commit, date)
}
