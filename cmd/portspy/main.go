// Command portspy shows what is listening on your local ports — with the
// project and command behind each — and kills it with one key.
package main

import (
	"os"

	"github.com/agenticraptor/portspy/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
