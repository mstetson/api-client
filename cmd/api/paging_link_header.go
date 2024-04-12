package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gonuts/commander"
	"github.com/peterhellberg/link"
)

var linkHeaderPagingCommands = []*commander.Command{
	{
		UsageLine: "get-paged [-max n] <url>",
		Short:     "make a paged GET request, relative to the API base URL",
		Flag:      *flag.NewFlagSet("get-paged", flag.ExitOnError),
		Run:       runLinkHeaderPagingGetPaged,
	},
}

func init() {
	linkHeaderPagingCommands[0].Flag.Int("max", 5, "max number of pages to load (0 is unlimited)")
}

func runLinkHeaderPagingGetPaged(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		cmd.Usage()
		return fmt.Errorf("wrong number of arguments, got %d want 1", len(args))
	}
	max := cmd.Lookup("max").(int)
	prefix := "["
	pages := 0
	nextURL := args[0]
	for nextURL != "" && (max <= 0 || pages < max) {
		_, err := os.Stdout.WriteString(prefix)
		if err != nil {
			return err
		}
		prefix = ","
		req, err := config.newRequest(cmd.Context(), "GET", nextURL, nil)
		if err != nil {
			return err
		}
		resp, err := config.doRequest(req, os.Stdout)
		if err != nil {
			return err
		}
		links := link.ParseResponse(resp)
		nextURL = ""
		for _, l := range links {
			if l.Rel == "next" {
				nextURL = l.URI
			}
		}
		pages++
	}
	_, err := os.Stdout.WriteString("]\n")
	return err
}
