package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gonuts/commander"
)

type JSONPagingConfig struct {
	NextPageURL string
}

type JSONPagingData struct {
	BaseURL string
	LastURL string
	Body    interface{} // parsed JSON body of prior request
	Data    map[string]any
}

var jsonPagingCommands = []*commander.Command{
	{
		UsageLine: "get-paged [-max n] <url>",
		Short:     "make a paged GET request, relative to the API base URL",
		Flag:      *flag.NewFlagSet("get-paged", flag.ExitOnError),
		Run:       runJSONPagingGetPaged,
	},
}

func init() {
	jsonPagingCommands[0].Flag.Int("max", 5, "max number of pages to load (0 is unlimited)")
}

func runJSONPagingGetPaged(cmd *commander.Command, args []string) error {
	if config.JSONPaging == nil {
		return fmt.Errorf("json paging not configured")
	}
	if config.JSONPaging.NextPageURL == "" {
		return fmt.Errorf("missing NextPageURL in json paging config")
	}
	if len(args) != 1 {
		cmd.Usage()
		return fmt.Errorf("wrong number of arguments, got %d want 1", len(args))
	}
	max := cmd.Lookup("max").(int)
	var buf bytes.Buffer
	data := JSONPagingData{
		BaseURL: args[0],
		LastURL: args[0],
		Data:    config.Data,
	}
	prefix := "["
	pages := 0
	for data.LastURL != "" && (max <= 0 || pages < max) {
		_, err := os.Stdout.WriteString(prefix)
		if err != nil {
			return err
		}
		prefix = ","
		req, err := config.newRequest(cmd.Context(), "GET", data.LastURL, nil)
		if err != nil {
			return err
		}
		err = config.doRequest(req, &buf)
		if err != nil {
			return err
		}
		data.Body = nil
		err = json.Unmarshal(buf.Bytes(), &data.Body)
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(os.Stdout)
		if err != nil {
			return err
		}
		data.LastURL, err = templateString(config.JSONPaging.NextPageURL, data)
		if err != nil {
			return err
		}
		data.LastURL = strings.TrimSpace(data.LastURL)
		pages++
	}
	_, err := os.Stdout.WriteString("]\n")
	return err
}
