package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gonuts/commander"
)

type Command struct {
	UsageLine   string
	Short       string
	Long        string
	Flag        []CommandFlag
	RequireArgs int

	Method   string
	URL      string
	Header   map[string]string
	ReadBody bool
	Body     string
}

type CommandFlag struct {
	Type           string
	Name           string
	Usage          string
	DefaultBool    bool
	DefaultFloat64 float64
	DefaultInt64   int64
	DefaultString  string
	DefaultUint64  uint64
}

var defaultCommands = []*Command{
	{
		UsageLine:   "delete <url>",
		Short:       "make a DELETE request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "DELETE",
		URL:         "{{ index .Flag.Args 0 }}",
	},
	{
		UsageLine:   "get <url>",
		Short:       "make a GET request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "GET",
		URL:         "{{ index .Flag.Args 0 }}",
	},
	{
		UsageLine:   "head <url>",
		Short:       "make a HEAD request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "HEAD",
		URL:         "{{ index .Flag.Args 0 }}",
	},
	{
		UsageLine:   "post <url>",
		Short:       "make a POST request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "POST",
		URL:         "{{ index .Flag.Args 0 }}",
		ReadBody:    true,
	},
	{
		UsageLine:   "put <url>",
		Short:       "make a PUT request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "PUT",
		URL:         "{{ index .Flag.Args 0 }}",
		ReadBody:    true,
	},
}

func (c *Command) Run(cmd *commander.Command, args []string) error {
	if c.RequireArgs >= 0 && len(args) != c.RequireArgs {
		cmd.Usage()
		return fmt.Errorf("wrong number of arguments, got %d want %d", len(args), c.RequireArgs)
	}
	d, err := c.processTemplates(cmd, args)
	if err != nil {
		return err
	}
	urlStr, err := config.relativeURLString(d.URL)
	if err != nil {
		return err
	}
	var body io.Reader
	if c.ReadBody {
		body = os.Stdin
	} else if d.Body != "" {
		body = strings.NewReader(d.Body)
	}
	req, err := http.NewRequestWithContext(cmd.Context(), d.Method, urlStr, body)
	if err != nil {
		return err
	}
	if config.UserAgent != "" {
		req.Header.Set("User-Agent", config.UserAgent)
	}
	for k, v := range d.Header {
		req.Header.Set(k, v)
	}
	return config.doRequest(req, os.Stdout)
}

type processedData struct {
	Method string
	URL    string
	Header map[string]string
	Body   string
}

type templateData struct {
	Flag *flag.FlagSet
	Data map[string]any
}

func (c *Command) processTemplates(cmd *commander.Command, args []string) (processedData, error) {
	var err error
	tdata := templateData{
		Flag: &cmd.Flag,
		Data: config.Data,
	}
	tmpl := func(s string) string {
		if err != nil {
			return ""
		}
		s, err = templateString(s, tdata)
		return s
	}
	d := processedData{
		Method: tmpl(c.Method),
		URL:    tmpl(c.URL),
		Header: make(map[string]string, len(c.Header)),
		Body:   tmpl(c.Body),
	}
	for k, v := range c.Header {
		d.Header[k] = tmpl(v)
	}
	return d, err
}

func (c *Command) Commander() (*commander.Command, error) {
	cmd := &commander.Command{
		UsageLine: c.UsageLine,
		Short:     c.Short,
		Long:      c.Long,
		Run:       c.Run,
	}
	cmd.Flag = *flag.NewFlagSet(cmd.FullName(), flag.ExitOnError)
	for _, f := range c.Flag {
		switch f.Type {
		case "bool":
			cmd.Flag.Bool(f.Name, f.DefaultBool, f.Usage)
		case "float64":
			cmd.Flag.Float64(f.Name, f.DefaultFloat64, f.Usage)
		case "int64":
			cmd.Flag.Int64(f.Name, f.DefaultInt64, f.Usage)
		case "string":
			cmd.Flag.String(f.Name, f.DefaultString, f.Usage)
		case "uint64":
			cmd.Flag.Uint64(f.Name, f.DefaultUint64, f.Usage)
		default:
			return nil, fmt.Errorf("%s: bad flag type %s", cmd.FullName(), f.Type)
		}
	}
	return cmd, nil
}
