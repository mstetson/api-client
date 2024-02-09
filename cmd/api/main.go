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

	"github.com/gonuts/commander"

	"bitbucket.org/classroomsystems/api-cli/apiconfig"
)

var cmd = &commander.Command{
	UsageLine: "api [-c config] command",
	Short:     "HTTP API CLI",
	Subcommands: []*commander.Command{
		deleteCommand,
		docsCommand,
		getCommand,
		postCommand,
		putCommand,
	},
}

var configName = flag.String("c", "", "API name for configuration")

type Config struct {
	Auth               string
	BaseURL            string
	DocsURL            string
	DefaultContentType string

	BasicAuth *struct {
		Username string
		Password string
	}
}

var config Config
var authState *apiconfig.AuthState

func main() {
	flag.Parse()
	var err error
	authState, err = apiconfig.Load(&config, *configName)
	if err != nil {
		if *configName == "" && errors.As(err, &apiconfig.ErrNotFound{}) {
			// the default config is fine
		} else {
			log.Println(err)
			os.Exit(1)
		}
	}
	err = cmd.Dispatch(context.Background(), flag.Args())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

func (c *Config) httpClient() (Client, error) {
	switch c.Auth {
	case "basic":
		return c.basicAuthClient()
	case "":
		return http.DefaultClient, nil
	default:
		return nil, fmt.Errorf("unknown authorization type: %s", c.Auth)
	}
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

func (c *Config) doRequest(req *http.Request) error {
	if req.Header.Get("Accept") == "" {
		req.Header.Add("Accept", c.DefaultContentType)
	}
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", c.DefaultContentType)
	}
	client, err := c.httpClient()
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Write(os.Stderr)
		return fmt.Errorf("HTTP error %s", resp.Status)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}
