package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gonuts/commander"
)

var postCommand = &commander.Command{
	Run:       runPost,
	UsageLine: "post <url>",
	Short:     "make a POST request, relative to the API base URL",
}

func runPost(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("post requires an URL parameter")
	}
	urlStr, err := config.relativeURLString(args[0])
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(cmd.Context(), "POST", urlStr, os.Stdin)
	if err != nil {
		return err
	}
	return config.doRequest(req)
}
