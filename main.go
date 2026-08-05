// SPDX-License-Identifier: MIT

package main

import "github.com/tenthirtyam/artifactory-content-library/internal/cli"

// Set by GoReleaser via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersion(version, commit, date)
	cli.Execute()
}
