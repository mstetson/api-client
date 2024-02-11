package main

import (
	"fmt"
	"net/http"

	"bitbucket.org/classroomsystems/api-cli/apiconfig"
)

type BasicAuthConfig struct {
	Username string
	Password string
}

func (c *Config) basicAuthClient() (Client, error) {
	if c.BasicAuth == nil {
		return nil, fmt.Errorf("basic auth not configured")
	}
	user, err := apiconfig.Deref(c.BasicAuth.Username)
	if err != nil {
		return nil, fmt.Errorf("BasicAuth.Username: %w", err)
	}
	pass, err := apiconfig.Deref(c.BasicAuth.Password)
	if err != nil {
		return nil, fmt.Errorf("BasicAuth.Password: %w", err)
	}
	return basicAuthClient{
		Client:   http.DefaultClient,
		Username: user,
		Password: pass,
	}, nil
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
