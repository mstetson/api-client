package main

import (
	"fmt"
	"net/http"

	"bitbucket.org/classroomsystems/api-cli/apiconfig"
)

type BearerAuthConfig struct {
	Prefix   string // Leave blank for "Bearer"
	Token    string
	NoPrefix bool
}

func newBearerAuthClient(c *Config, a *apiconfig.AuthState) (Client, error) {
	if c.BearerAuth == nil {
		return nil, fmt.Errorf("bearer auth not configured")
	}
	if c.BearerAuth.Prefix == "" {
		c.BearerAuth.Prefix = "Bearer"
	}
	client := bearerAuthClient{Client: http.DefaultClient}
	var deref apiconfig.Dereffer
	if c.BearerAuth.NoPrefix {
		client.Authorization = deref.String(c.BearerAuth.Token)
	} else {
		client.Authorization = deref.String(c.BearerAuth.Prefix) +
			" " + deref.String(c.BearerAuth.Token)
	}
	return client, deref.Error
}

type bearerAuthClient struct {
	Client        *http.Client
	Authorization string
}

func (c bearerAuthClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", c.Authorization)
	return c.Client.Do(req)
}
