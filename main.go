package main

import (
	"fmt"
	"os"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/client"
)

func main() {
	opts, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "httpcat: %v\n", err)
		fmt.Fprintf(os.Stderr, "Try 'httpcat --help' for usage.\n")
		os.Exit(1)
	}

	if err := client.Do(opts); err != nil {
		fmt.Fprintf(os.Stderr, "httpcat: %v\n", err)
		os.Exit(1)
	}
}
