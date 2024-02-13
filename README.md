# API CLI

I interact with HTTP APIs all the time, so I need a decent tool for them. I got tired of using Postman and Insomnia: they are ecosystems unto themselves, and they aren't made for the kind of scripting and exploration I tend to do.

## Goals of this tool

- Play nice in shell scripts.
- Play nice with version control.
- Retrieve secrets from 1Password. It would be easy to add other secret stores, but I don't need that yet.
- Handle all the kinds of authentication I need.
- Work with just about any HTTP API.
- Help me get to API docs when I need them.
- Let me store specific requests so I can use or refer to them.
- Load config based on the file system hierarchy, so I can create workspaces by creating directories.
- Stay small and don't get ambitious.

## Installation

```sh
go install bitbucket.org/classroomsystems/api-cli/cmd/api@latest
```

## Example Config

Here's a simple configuration file for the Agilix Buzz API:

```toml
# Agilix Buzz API
Auth = "query"
BaseURL = "https://api.agilixbuzz.com/cmd/"
DocsURL = "https://api.agilixbuzz.com/docs/"

[QueryAuth]
"_token" = "{{op://Micah at Work/DLAP Admin User/credential}}"
```

With that in a file named `api-dlap.config`, then in any subdir of that place, I can do stuff like this:

```
$ api -c dlap help
api - HTTP API CLI (dlap)

Commands:

    delete      make a DELETE request, relative to the API base URL
    docs        open documentation web site
    get         make a GET request, relative to the API base URL
    head        make a HEAD request, relative to the API base URL
    post        make a POST request, relative to the API base URL
    put         make a PUT request, relative to the API base URL

Use "api help <command>" for more information about a command.

$ api -c dlap get 'getdomain2?domainid=//staff'| jsonfmt
{
	"response": {
		"code": "OK",
		"domain": {
			"id": "190011616",
			"name": "SchoolsPLP Staff",
			"userspace": "staff",
			"parentid": "62643388",
			"reference": "",
			"guid": "c0c43380-0a4e-4c01-b8bc-38fbacf180ce",
			"flags": 0,
			"creationdate": "2023-03-08T15:19:12.977Z",
			"creationby": "114151265",
			"modifieddate": "2023-05-23T02:00:22.61Z",
			"modifiedby": "63053543",
			"version": "3"
		}
	}
}
```

This exact file won't work for you, since the authentication token is being retrieved by reference from one of my 1Password vaults. When I use the API this way, 1Password prompts me to authorize the CLI with my fingerprint, granting it access to the secrets for a time.

Running `api -c dlap docs` opens Agilix's API documentation web site in my browser.

## Other stuff

Poke at the code, it's not meant to be a black box. There are several kinds of auth supported. You can add new subcommands via the configuration file. These can construct requests by applying Go templates to configuration data and command-line arguments.

## Bugs

- No tests
- Poor docs
- Only 1Password CLI for secret retrieval
- Web page launches use plan9port's web script
- Likely others