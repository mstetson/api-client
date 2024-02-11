package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/gonuts/commander"
)

type Command struct {
	UsageLine   string
	Short       string
	Long        string
	Flag        []CommandFlag
	RequireArgs int

	Method          string
	URL             string
	Header          map[string]string
	HasBody         bool
	RequestCommand  string
	ResponseCommand string
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
		URL:         "{{ index .Args 0 }}",
	},
	{
		UsageLine:   "get <url>",
		Short:       "make a GET request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "GET",
		URL:         "{{ index .Args 0 }}",
	},
	{
		UsageLine:   "head <url>",
		Short:       "make a HEAD request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "HEAD",
		URL:         "{{ index .Args 0 }}",
	},
	{
		UsageLine:   "post <url>",
		Short:       "make a POST request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "POST",
		URL:         "{{ index .Args 0 }}",
		HasBody:     true,
	},
	{
		UsageLine:   "put <url>",
		Short:       "make a PUT request, relative to the API base URL",
		RequireArgs: 1,
		Method:      "PUT",
		URL:         "{{ index .Args 0 }}",
		HasBody:     true,
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
	if c.HasBody {
		body = os.Stdin
	}
	if d.RequestCommand != "" {
		panic("TODO")
	}
	req, err := http.NewRequestWithContext(cmd.Context(), d.Method, urlStr, body)
	if err != nil {
		return err
	}
	for k, v := range d.Header {
		req.Header.Set(k, v)
	}
	if d.ResponseCommand != "" {
		panic("TODO")
	}
	return config.doRequest(req, os.Stdout)
}

type processedData struct {
	Method          string
	URL             string
	Header          map[string]string
	RequestCommand  string
	ResponseCommand string
}

func (c *Command) processTemplates(cmd *commander.Command, args []string) (processedData, error) {
	var err error
	tmpl := func(s string) string {
		var t *template.Template
		if err != nil {
			return ""
		}
		t, err = template.New("").Parse(s)
		if err != nil {
			return ""
		}
		var b strings.Builder
		err = t.Execute(&b, &cmd.Flag)
		return b.String()
	}
	d := processedData{
		Method:          tmpl(c.Method),
		URL:             tmpl(c.URL),
		RequestCommand:  tmpl(c.RequestCommand),
		ResponseCommand: tmpl(c.ResponseCommand),
		Header:          make(map[string]string, len(c.Header)),
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
