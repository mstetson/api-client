package main

import (
	"fmt"
	"net/http"

	"github.com/mstetson/api-client/apiconfig"
)

type BasicAuthConfig struct {
	Username string
	Password string
}

func newBasicAuthClient(c *Config, a *apiconfig.AuthState) (Client, error) {
	if c.BasicAuth == nil {
		return nil, fmt.Errorf("basic auth not configured")
	}
	var deref apiconfig.Dereffer
	return basicAuthClient{
		Client:   http.DefaultClient,
		Username: deref.String(c.BasicAuth.Username),
		Password: deref.String(c.BasicAuth.Password),
	}, deref.Error
}

type basicAuthClient struct {
	Client   *http.Client
	Username string
	Password string
}

func (c basicAuthClient) Do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.Username, c.Password)
	return c.Client.Do(req)
}
