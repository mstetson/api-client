package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gonuts/commander"
)

var putCommand = &commander.Command{
	Run:       runPut,
	UsageLine: "put <url>",
	Short:     "make a PUT request, relative to the API base URL",
}

func runPut(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("put requires an URL parameter")
	}
	urlStr, err := config.relativeURLString(args[0])
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(cmd.Context(), "PUT", urlStr, os.Stdin)
	if err != nil {
		return err
	}
	return config.doRequest(req)
}
