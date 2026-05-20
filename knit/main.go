// Command knit is a CLI for installing, uninstalling, and listing
// knowledge packs in each AI coding tool's configuration directory.
//
// This main.go is a thin wrapper around internal/cli and contains no
// business logic. Subcommand routing, flags, I/O, and exit-code
// decisions are all centralized in internal/cli.
package main

import (
	"context"
	"os"

	"github.com/theoden9014/ai-knowledge-base/knit/internal/cli"
)

// version is expected to be injected at build time via
// -ldflags "-X main.version=<value>". If left empty, such as when built
// with go install, cli.Execute treats it as "(devel)".
var version = ""

func main() {
	os.Exit(cli.Execute(context.Background(), nil, "knit", version).Int())
}
