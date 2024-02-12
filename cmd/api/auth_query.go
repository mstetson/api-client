package main

import (
	"fmt"
	"net/http"

	"bitbucket.org/classroomsystems/api-cli/apiconfig"
)

type QueryAuthConfig map[string]string

func (c *Config) queryAuthClient() (Client, error) {
	if c.QueryAuth == nil {
		return nil, fmt.Errorf("query auth not configured")
	}
	var deref apiconfig.Dereffer
	return queryAuthClient{
		Client: http.DefaultClient,
		Config: deref.StringMap(c.QueryAuth),
	}, deref.Error
}

type queryAuthClient struct {
	Client *http.Client
	Config QueryAuthConfig
}

func (c queryAuthClient) Do(req *http.Request) (*http.Response, error) {
	q := req.URL.Query()
	for k, v := range c.Config {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return c.Client.Do(req)
}
