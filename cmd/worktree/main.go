package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var Version = "dev"

func main() {
	flags := flag.NewFlagSet(fmt.Sprintf("%s @ %s", filepath.Base(os.Args[0]), Version), flag.ExitOnError)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		_, _ = fmt.Fprintf(flags.Output(), "%s [args ...]\n", filepath.Base(os.Args[0]))
		_, _ = fmt.Fprintln(flags.Output(), "More details here.")
		flags.PrintDefaults()
	}
	_ = flags.Parse(os.Args[1:])

	fmt.Println("Hello, world!")
}
