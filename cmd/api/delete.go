package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gonuts/commander"
)

var deleteCommand = &commander.Command{
	Run:       runDelete,
	UsageLine: "delete <url>",
	Short:     "make a DELETE request, relative to the API base URL",
}

func runDelete(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("delete requires an URL parameter")
	}
	urlStr, err := config.relativeURLString(args[0])
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(cmd.Context(), "DELETE", urlStr, os.Stdin)
	if err != nil {
		return err
	}
	return config.doRequest(req)
}
