package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/gonuts/commander"

	"github.com/mstetson/api-client/apiconfig"
)

var cmd = &commander.Command{
	UsageLine: "api [-c config] command",
	Short:     "HTTP API CLI",
}

var configName = flag.String("c", "", "API name for configuration")

var config Config
var authState *apiconfig.AuthState

type Config struct {
	Auth               string
	Paging             string
	BaseURL            string
	DocsURL            string
	DefaultContentType string
	UserAgent          string

	BasicAuth  *BasicAuthConfig
	BearerAuth *BearerAuthConfig
	OAuth1     *OAuth1Config
	OAuth2     *OAuth2Config
	QueryAuth  QueryAuthConfig

	JSONPaging *JSONPagingConfig

	Data map[string]any

	Command []*Command
}

var authTypeClients = map[string]func(*Config, *apiconfig.AuthState) (Client, error){
	"basic":  newBasicAuthClient,
	"bearer": newBearerAuthClient,
	"oauth1": newOAuth1Client,
	"oauth2": newOAuth2Client,
	"query":  newQueryAuthClient,
}

var authCommands = map[string][]*commander.Command{
	"oauth1": oauth1Commands,
	"oauth2": oauth2Commands,
}

var pagingCommands = map[string][]*commander.Command{
	"json":        jsonPagingCommands,
	"link-header": linkHeaderPagingCommands,
}

func main() {
	flag.Parse()
	var err error
	if *configName != "" {
		cmd.Short = fmt.Sprintf("HTTP API CLI (%s)", *configName)
	}
	config.DefaultContentType = "application/json"
	authState, err = apiconfig.Load(&config, *configName)
	if err != nil {
		if *configName == "" && errors.As(err, &apiconfig.ErrNotFound{}) {
			// the default config is fine
		} else {
			log.Println(err)
			os.Exit(1)
		}
	}
	err = config.addCommands(cmd)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = cmd.Dispatch(context.Background(), flag.Args())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func commandName() string {
	if *configName == "" {
		return "api"
	}
	return "api -c " + *configName
}

func launchBrowser(urlStr string) error {
	return exec.Command("web", urlStr).Run()
}

func (c *Config) addCommands(cmd *commander.Command) error {
	if config.DocsURL != "" {
		cmd.Subcommands = append(cmd.Subcommands, docsCommand)
	}
	if cmds := authCommands[c.Auth]; cmds != nil {
		cmd.Subcommands = append(cmd.Subcommands, cmds...)
	}
	if cmds := pagingCommands[c.Paging]; cmds != nil {
		cmd.Subcommands = append(cmd.Subcommands, cmds...)
	}
	subs := append(defaultCommands, c.Command...)
	for _, s := range subs {
		sub, err := s.Commander()
		if err != nil {
			return err
		}
		cmd.Subcommands = append(cmd.Subcommands, sub)
	}
	return nil
}

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

func (c *Config) httpClient() (Client, error) {
	if c.Auth == "" {
		return http.DefaultClient, nil
	}
	fn, ok := authTypeClients[c.Auth]
	if !ok {
		return nil, fmt.Errorf("unknown authorization type: %s", c.Auth)
	}
	return fn(c, authState)
}

func (c *Config) newRequest(ctx context.Context, method, urlStr string, body io.Reader) (*http.Request, error) {
	u, err := c.relativeURLString(urlStr)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if config.UserAgent != "" {
		req.Header.Set("User-Agent", config.UserAgent)
	}
	return req, nil
}

func (c *Config) relativeURLString(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	if c.BaseURL == "" {
		return u.String(), nil
	}
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(u).String(), nil
}

func (c *Config) doRequest(req *http.Request, out io.Writer) (resp *http.Response, err error) {
	if req.Header.Get("Accept") == "" {
		req.Header.Add("Accept", c.DefaultContentType)
	}
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", c.DefaultContentType)
	}
	client, err := c.httpClient()
	if err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Write(os.Stderr)
		return resp, fmt.Errorf("HTTP error %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

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
	return launchBrowser(config.DocsURL)
}

func templateString(tmpl string, data any) (string, error) {
	t, err := template.New("").Funcs(templateFuncs).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = t.Execute(&b, data)
	return b.String(), err
}

var templateFuncs = template.FuncMap{
	"setQueryParam": func(urlStr, key, value string) (string, error) {
		u, err := url.Parse(urlStr)
		if err != nil {
			return "", err
		}
		q := u.Query()
		q.Set(key, value)
		u.RawQuery = q.Encode()
		return u.String(), nil
	},
}
