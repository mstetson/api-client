package main

import (
	"fmt"
	"os/exec"

	"github.com/gonuts/commander"
)

var docsCommand = &commander.Command{
	Run:       runDocs,
	UsageLine: "docs",
	Short:     "open documentation web site",
}

func runDocs(cmd *commander.Command, args []string) error {
	if config.DocsURL == "" {
		fmt.Println("no docs defined")
		return nil
	}
	c := exec.CommandContext(cmd.Context(), "web", config.DocsURL)
	return c.Run()
}
