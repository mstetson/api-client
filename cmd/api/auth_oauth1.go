package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/classroomsystems/api-cli/apiconfig"
	"github.com/gonuts/commander"
	"github.com/mrjones/oauth"
)

type OAuth1Config struct {
	// Consumer Info
	ConsumerKey    string
	ConsumerSecret string

	//CallbackURL string // Do service providers allow localhost callback URLs?

	// Service Provider Info
	RequestTokenURL   string
	AuthorizeTokenURL string
	AccessTokenURL    string
	AccessDuration    time.Duration

	// Less Common Options
	HttpMethod                       string
	BodyHash                         bool
	IgnoreTimestamp                  bool
	SignQueryParams                  bool
	AdditionalParams                 map[string]string
	AdditionalAuthorizationURLParams map[string]string
	AuthorizeTokenURLTemplate        string
}

func (c *OAuth1Config) deref() (*OAuth1Config, error) {
	var deref apiconfig.Dereffer
	return &OAuth1Config{
		ConsumerKey:    deref.String(c.ConsumerKey),
		ConsumerSecret: deref.String(c.ConsumerSecret),

		RequestTokenURL:   deref.String(c.RequestTokenURL),
		AuthorizeTokenURL: deref.String(c.AuthorizeTokenURL),
		AccessTokenURL:    deref.String(c.AccessTokenURL),
		AccessDuration:    c.AccessDuration,

		HttpMethod:                       deref.String(c.HttpMethod),
		BodyHash:                         c.BodyHash,
		IgnoreTimestamp:                  c.IgnoreTimestamp,
		SignQueryParams:                  c.SignQueryParams,
		AdditionalParams:                 deref.StringMap(c.AdditionalParams),
		AdditionalAuthorizationURLParams: deref.StringMap(c.AdditionalAuthorizationURLParams),
		AuthorizeTokenURLTemplate:        deref.String(c.AuthorizeTokenURLTemplate),
	}, deref.Error
}

var oauth1Commands = []*commander.Command{
	{
		UsageLine: "auth [-reset] [verification-code]",
		Short:     "do OAuth 1.0 authorization",
		Flag:      *flag.NewFlagSet("auth", flag.ExitOnError),
		Run:       runOAuth1,
	},
}

func init() {
	oauth1Commands[0].Flag.Bool("reset", false, "ignore existing credentials and start over")
}

func runOAuth1(cmd *commander.Command, args []string) error {
	if config.OAuth1 == nil {
		return fmt.Errorf("oauth1 not configured")
	}
	c, err := newOAuth1ClientConcrete(config.OAuth1, authState)
	if err != nil {
		return err
	}
	if cmd.Lookup("reset").(bool) {
		err := c.resetAuth()
		if err != nil {
			return err
		}
	}
	if c.isLoggedIn() {
		fmt.Println("access token is current")
		return nil
	}
	switch len(args) {
	case 0:
		accessURL, err := c.requestAccess()
		if err != nil {
			return err
		}
		fmt.Println("URL:", accessURL)
		fmt.Println("verify access with", commandName(), "auth CODE")
		return launchBrowser(accessURL)
	case 1:
		err := c.verifyAccess(args[0])
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	default:
		cmd.Usage()
		return nil
	}
}

func newOAuth1Client(c *Config, a *apiconfig.AuthState) (Client, error) {
	if c.OAuth1 == nil {
		return nil, fmt.Errorf("oauth1 not configured")
	}
	cl, err := newOAuth1ClientConcrete(c.OAuth1, a)
	return cl, err
}

type oauth1Client struct {
	config           *OAuth1Config
	auth             *apiconfig.AuthState
	consumer         *oauth.Consumer
	accessToken      *oauth.AccessToken
	accessTokenTime  time.Time
	requestToken     *oauth.RequestToken
	requestTokenTime time.Time
	client           *http.Client
}

func newOAuth1ClientConcrete(config *OAuth1Config, auth *apiconfig.AuthState) (*oauth1Client, error) {
	config, err := config.deref()
	if err != nil {
		return nil, err
	}
	c := &oauth1Client{
		config: config,
		auth:   auth,
	}
	c.consumer = c.getOAuthConsumer()
	c.accessToken, c.accessTokenTime = c.currentAccessToken()
	c.requestToken, c.requestTokenTime = c.currentRequestToken()
	if c.isLoggedIn() {
		c.client, err = c.consumer.MakeHttpClient(c.accessToken)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *oauth1Client) Do(req *http.Request) (*http.Response, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not logged in: try %s auth", commandName())
	}
	return c.client.Do(req)
}

func (c *oauth1Client) isLoggedIn() bool {
	if c.accessToken == nil {
		return false
	}
	if c.config.AccessDuration > 0 && time.Since(c.accessTokenTime) > c.config.AccessDuration {
		return false
	}
	return true
}

func (c *oauth1Client) resetAuth() error {
	c.accessToken = nil
	c.accessTokenTime = time.Time{}
	delete(c.auth.Values, "AccessToken")
	delete(c.auth.Values, "AccessTokenSecret")
	delete(c.auth.Values, "AccessTokenDate")
	c.requestToken = nil
	c.requestTokenTime = time.Time{}
	delete(c.auth.Values, "RequestToken")
	delete(c.auth.Values, "RequestTokenSecret")
	delete(c.auth.Values, "RequestTokenDate")
	return c.auth.Save()
}

func (c *oauth1Client) requestAccess() (accessURL string, err error) {
	c.requestToken, accessURL, err = c.consumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		return "", err
	}
	c.requestTokenTime = time.Now()
	c.accessToken = nil
	c.accessTokenTime = time.Time{}

	delete(c.auth.Values, "AccessToken")
	delete(c.auth.Values, "AccessTokenSecret")
	delete(c.auth.Values, "AccessTokenDate")
	c.auth.Values["RequestToken"] = c.requestToken.Token
	c.auth.Values["RequestTokenSecret"] = c.requestToken.Secret
	c.auth.Values["RequestTokenDate"] = c.requestTokenTime.Format(time.RFC3339)

	if c.config.AuthorizeTokenURLTemplate != "" {
		tdata := struct {
			Config       *OAuth1Config
			RequestToken *oauth.RequestToken
		}{
			Config:       c.config,
			RequestToken: c.requestToken,
		}
		accessURL, err = templateString(c.config.AuthorizeTokenURLTemplate, tdata)
		if err != nil {
			return "", err
		}
	}

	return accessURL, c.auth.Save()
}

func (c *oauth1Client) verifyAccess(code string) error {
	var err error
	c.accessToken, err = c.consumer.AuthorizeToken(c.requestToken, code)
	if err != nil {
		return err
	}
	c.accessTokenTime = time.Now()
	c.requestToken = nil
	c.requestTokenTime = time.Time{}

	delete(c.auth.Values, "RequestToken")
	delete(c.auth.Values, "RequestTokenSecret")
	delete(c.auth.Values, "RequestTokenDate")
	c.auth.Values["AccessToken"] = c.accessToken.Token
	c.auth.Values["AccessTokenSecret"] = c.accessToken.Secret
	c.auth.Values["AccessTokenDate"] = c.accessTokenTime.Format(time.RFC3339)
	err = c.auth.Save()
	if err != nil {
		return err
	}
	c.client, err = c.consumer.MakeHttpClient(c.accessToken)
	return err
}

func (c *oauth1Client) getOAuthConsumer() *oauth.Consumer {
	oc := oauth.NewConsumer(
		c.config.ConsumerKey,
		c.config.ConsumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   c.config.RequestTokenURL,
			AuthorizeTokenUrl: c.config.AuthorizeTokenURL,
			AccessTokenUrl:    c.config.AccessTokenURL,
			HttpMethod:        c.config.HttpMethod,
			BodyHash:          c.config.BodyHash,
			IgnoreTimestamp:   c.config.IgnoreTimestamp,
			SignQueryParams:   c.config.SignQueryParams,
		})
	oc.AdditionalParams = c.config.AdditionalParams
	oc.AdditionalAuthorizationUrlParams = c.config.AdditionalAuthorizationURLParams
	return oc
}

func (c *oauth1Client) currentAccessToken() (*oauth.AccessToken, time.Time) {
	if c.auth.Values["AccessTokenDate"] == "" {
		return nil, time.Time{}
	}
	dt, _ := time.Parse(time.RFC3339, c.auth.Values["AccessTokenDate"])
	if c.config.AccessDuration > 0 && time.Since(dt) > c.config.AccessDuration {
		return nil, time.Time{}
	}
	return &oauth.AccessToken{
		Token:  c.auth.Values["AccessToken"],
		Secret: c.auth.Values["AccessTokenSecret"],
	}, dt
}

func (c *oauth1Client) currentRequestToken() (*oauth.RequestToken, time.Time) {
	if c.auth.Values["RequestTokenDate"] == "" {
		return nil, time.Time{}
	}
	dt, _ := time.Parse(time.RFC3339, c.auth.Values["RequestTokenDate"])
	if time.Since(dt) > 5*time.Minute {
		return nil, time.Time{}
	}
	return &oauth.RequestToken{
		Token:  c.auth.Values["RequestToken"],
		Secret: c.auth.Values["RequestTokenSecret"],
	}, dt
}
