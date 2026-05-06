package main

import (
	"fmt"
	"os"

	"github.com/kborup-redhat/pq-notes/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
