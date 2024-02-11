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
	return queryAuthClient{
		Client: http.DefaultClient,
		Config: c.QueryAuth,
	}, nil
}

type queryAuthClient struct {
	Client *http.Client
	Config QueryAuthConfig
}

func (c queryAuthClient) Do(req *http.Request) (*http.Response, error) {
	q := req.URL.Query()
	for k, v := range c.Config {
		v, err := apiconfig.Deref(v)
		if err != nil {
			return nil, err
		}
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return c.Client.Do(req)
}
