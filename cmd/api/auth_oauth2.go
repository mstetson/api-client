package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gonuts/commander"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/mstetson/api-client/apiconfig"
)

type OAuth2Config struct {
	GrantType string // AuthorizationCode (default), ClientCredentials, PasswordCredentials

	// Used by all grant types
	ClientID       string
	ClientSecret   string
	Scopes         []string
	TokenURL       string
	TokenURLParams url.Values
	AuthStyle      string // AutoDetect (default), InParams, InHeader
	TokenType      string // empty is the same as "Bearer"

	// Used by GrantType == AuthorizationCode
	UsePKCE       bool
	RedirectURL   string
	AuthURL       string
	AuthURLParams url.Values
	State         string // defaults to "state" if blank

	// Used by GrantType == PasswordCredentials
	Username string
	Password string
}

func (c *OAuth2Config) deref() (*OAuth2Config, error) {
	var deref apiconfig.Dereffer
	return &OAuth2Config{
		GrantType: deref.String(c.GrantType),

		ClientID:       deref.String(c.ClientID),
		ClientSecret:   deref.String(c.ClientSecret),
		Scopes:         deref.StringSlice(c.Scopes),
		TokenURL:       deref.String(c.TokenURL),
		TokenURLParams: deref.URLValues(c.TokenURLParams),
		AuthStyle:      deref.String(c.AuthStyle),
		TokenType:      deref.String(c.TokenType),

		UsePKCE:       c.UsePKCE,
		RedirectURL:   deref.String(c.RedirectURL),
		AuthURL:       deref.String(c.AuthURL),
		AuthURLParams: deref.URLValues(c.AuthURLParams),
		State:         deref.String(c.State),

		Username: deref.String(c.Username),
		Password: deref.String(c.Password),
	}, deref.Error
}

func (c *OAuth2Config) authStyle() oauth2.AuthStyle {
	switch c.AuthStyle {
	case "InParams":
		return oauth2.AuthStyleInParams
	case "InHeader":
		return oauth2.AuthStyleInHeader
	case "", "AutoDetect":
		return oauth2.AuthStyleAutoDetect
	default:
		panic("unrecognized oauth2 AuthStyle: " + c.AuthStyle)
	}
}

var oauth2Commands = []*commander.Command{
	{
		UsageLine: "auth [-reset] [verification code or url]",
		Short:     "do OAuth 2.0 authorization",
		Flag:      *flag.NewFlagSet("auth", flag.ExitOnError),
		Run:       runOAuth2,
	},
}

func init() {
	oauth2Commands[0].Flag.Bool("reset", false, "ignore existing credentials and start over")
}

func runOAuth2(cmd *commander.Command, args []string) error {
	if config.OAuth2 == nil {
		return fmt.Errorf("oauth2 not configured")
	}
	c, err := newOAuth2ClientConcrete(config.OAuth2, authState)
	if err != nil {
		return err
	}
	if cmd.Lookup("reset").(bool) {
		err := c.resetAuth()
		if err != nil {
			return err
		}
	}
	if c.tokenSource != nil {
		token, err := c.tokenSource.Token()
		if err == nil && token.Valid() {
			fmt.Println("access token is current")
			return nil
		}
		c.tokenSource = nil
	}
	switch c.config.GrantType {
	case "", "AuthorizationCode":
		return c.authAuthCode(cmd, args)
	case "ClientCredentials":
		return c.authClientCreds(cmd, args)
	case "PasswordCredentials":
		return c.authPassword(cmd, args)
	default:
		return fmt.Errorf("unknown OAuth2 grant type: %s", c.config.GrantType)
	}
}

func newOAuth2Client(c *Config, a *apiconfig.AuthState) (Client, error) {
	if c.OAuth2 == nil {
		return nil, fmt.Errorf("oauth2 not configured")
	}
	cl, err := newOAuth2ClientConcrete(c.OAuth2, a)
	return cl, err
}

type oauth2Client struct {
	config      *OAuth2Config
	auth        *apiconfig.AuthState
	acConfig    *oauth2.Config
	ccConfig    *clientcredentials.Config
	tokenSource *oauth2TokenSource
	client      *http.Client
}

func newOAuth2ClientConcrete(config *OAuth2Config, auth *apiconfig.AuthState) (*oauth2Client, error) {
	config, err := config.deref()
	if err != nil {
		return nil, err
	}
	c := &oauth2Client{
		config: config,
		auth:   auth,
		client: http.DefaultClient,
	}
	c.translateConfig()
	c.tokenSource = c.newTokenSource(context.Background(), nil)
	c.tokenSource.loadToken()
	return c, nil
}

func (c *oauth2Client) Do(req *http.Request) (*http.Response, error) {
	if c.tokenSource == nil {
		if req.Body != nil {
			// http.Client.Do guarantees close, even on error.
			req.Body.Close()
		}
		return nil, fmt.Errorf("not logged in: try %s auth", commandName())
	}
	token, err := c.tokenSource.Token()
	if err != nil {
		if req.Body != nil {
			// http.Client.Do guarantees close, even on error.
			req.Body.Close()
		}
		return nil, err
	}
	token.SetAuthHeader(req)
	return c.client.Do(req)
}

func (c *oauth2Client) translateConfig() {
	if c.config.GrantType == "ClientCredentials" {
		c.ccConfig = &clientcredentials.Config{
			ClientID:       c.config.ClientID,
			ClientSecret:   c.config.ClientSecret,
			TokenURL:       c.config.TokenURL,
			Scopes:         c.config.Scopes,
			EndpointParams: c.config.TokenURLParams,
			AuthStyle:      c.config.authStyle(),
		}
	} else {
		c.acConfig = &oauth2.Config{
			ClientID:     c.config.ClientID,
			ClientSecret: c.config.ClientSecret,
			Scopes:       c.config.Scopes,
			RedirectURL:  c.config.RedirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:   c.config.AuthURL,
				TokenURL:  c.config.TokenURL,
				AuthStyle: c.config.authStyle(),
			},
		}
	}
}

func (c *oauth2Client) authAuthCode(cmd *commander.Command, args []string) error {
	switch len(args) {
	case 0:
		authCodeURL, err := c.authCodeURL()
		if err != nil {
			return err
		}
		fmt.Println("URL:", authCodeURL)
		fmt.Println("verify access with", commandName(), "auth CODE")
		return launchBrowser(authCodeURL)
	case 1:
		code := args[0]
		if strings.HasPrefix(args[0], "http") {
			u, err := url.Parse(args[0])
			if err != nil {
				return err
			}
			code = u.Query().Get("code")
			// TODO: check state?
			if scopes := u.Query().Get("scope"); scopes != "" {
				fmt.Println("Scopes:", scopes)
			}
		}
		err := c.exchangeCode(cmd.Context(), code)
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

func (c *oauth2Client) authClientCreds(cmd *commander.Command, args []string) error {
	ctx := cmd.Context()
	token, err := c.ccConfig.Token(ctx)
	if err != nil {
		return err
	}
	token.TokenType = c.config.TokenType
	c.tokenSource = c.newTokenSource(ctx, token)
	return c.tokenSource.saveToken()
}

func (c *oauth2Client) authPassword(cmd *commander.Command, args []string) error {
	ctx := cmd.Context()
	token, err := c.acConfig.PasswordCredentialsToken(ctx, c.config.Username, c.config.Password)
	if err != nil {
		return err
	}
	token.TokenType = c.config.TokenType
	c.tokenSource = c.newTokenSource(ctx, token)
	return c.tokenSource.saveToken()
}

func (c *oauth2Client) resetAuth() error {
	c.tokenSource = nil
	c.auth.Values = make(map[string]string)
	return c.auth.Save()
}

func (c *oauth2Client) authCodeURL() (u string, err error) {
	var opts []oauth2.AuthCodeOption
	for k, vs := range c.config.AuthURLParams {
		for _, v := range vs {
			opts = append(opts, oauth2.SetAuthURLParam(k, v))
		}
	}
	if c.config.UsePKCE {
		c.auth.Values["PKCEVerifier"] = oauth2.GenerateVerifier()
		opts = append(opts, oauth2.S256ChallengeOption(c.auth.Values["PKCEVerifier"]))
		err = c.auth.Save()
	}
	if c.config.State == "" {
		c.config.State = "state"
	}
	u = c.acConfig.AuthCodeURL(c.config.State, opts...)
	return u, err
}

func (c *oauth2Client) exchangeCode(ctx context.Context, code string) error {
	var opts []oauth2.AuthCodeOption
	for k, vs := range c.config.TokenURLParams {
		for _, v := range vs {
			opts = append(opts, oauth2.SetAuthURLParam(k, v))
		}
	}
	if c.config.UsePKCE {
		opts = append(opts, oauth2.VerifierOption(c.auth.Values["PKCEVerifier"]))
	}
	token, err := c.acConfig.Exchange(ctx, code, opts...)
	if err != nil {
		return err
	}
	token.TokenType = c.config.TokenType
	c.tokenSource = c.newTokenSource(ctx, token)
	return c.tokenSource.saveToken()
}

// This is like oauth2.ReuseTokenSource,
// but tokens are saved in auth for use across calls of the program.
type oauth2TokenSource struct {
	auth *apiconfig.AuthState
	new  oauth2.TokenSource // called when t is expired.

	mu sync.Mutex // guards t
	t  *oauth2.Token
}

func (c *oauth2Client) newTokenSource(ctx context.Context, token *oauth2.Token) *oauth2TokenSource {
	var ts oauth2.TokenSource
	if c.config.GrantType == "ClientCredentials" {
		ts = c.ccConfig.TokenSource(ctx)
	} else {
		ts = c.acConfig.TokenSource(ctx, nil)
	}
	return &oauth2TokenSource{
		auth: c.auth,
		new:  ts,
		t:    token,
	}
}

func (s *oauth2TokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t.Valid() {
		return s.t, nil
	}
	t, err := s.new.Token()
	if err != nil {
		return nil, err
	}
	s.t = t
	return t, s.saveToken()
}

func (s *oauth2TokenSource) saveToken() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth.Values["AccessToken"] = s.t.AccessToken
	s.auth.Values["TokenType"] = s.t.TokenType
	s.auth.Values["RefreshToken"] = s.t.RefreshToken
	s.auth.Values["Expiry"] = s.t.Expiry.Format(time.RFC3339)
	return s.auth.Save()
}

func (s *oauth2TokenSource) loadToken() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t = &oauth2.Token{
		AccessToken:  s.auth.Values["AccessToken"],
		TokenType:    s.auth.Values["TokenType"],
		RefreshToken: s.auth.Values["RefreshToken"],
	}
	if x, err := time.Parse(time.RFC3339, s.auth.Values["Expiry"]); err == nil {
		s.t.Expiry = x
	}
	if s.t.AccessToken == "" {
		s.t = nil
	}
}
