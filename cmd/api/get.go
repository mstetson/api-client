package main

import (
	"fmt"
	"net/http"

	"github.com/gonuts/commander"
)

var getCommand = &commander.Command{
	Run:       runGet,
	UsageLine: "get <url>",
	Short:     "make a GET request, relative to the API base URL",
}

func runGet(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("get requires an URL parameter")
	}
	urlStr, err := config.relativeURLString(args[0])
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(cmd.Context(), "GET", urlStr, nil)
	if err != nil {
		return err
	}
	return config.doRequest(req)
}
