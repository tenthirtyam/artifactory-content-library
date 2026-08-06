// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2026 Ryan Johnson

package main

import "github.com/tenthirtyam/artifactory-content-library/internal/cli"

// Set by GoReleaser via ldflags. When left as "dev", SetVersion falls back to
// runtime/debug.BuildInfo (e.g. go install ...@v0.1.0).
var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
