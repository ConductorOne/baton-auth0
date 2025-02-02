![Baton Logo](./docs/images/baton-logo.png)

# `baton-auth0` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-auth0.svg)](https://pkg.go.dev/github.com/conductorone/baton-auth0) ![main ci](https://github.com/conductorone/baton-auth0/actions/workflows/main.yaml/badge.svg)

`baton-auth0` is a connector for built using the [Baton SDK](https://github.com/conductorone/baton-sdk).

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-auth0
baton-auth0
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_DOMAIN_URL=domain_url -e BATON_API_KEY=apiKey -e BATON_USERNAME=username ghcr.io/conductorone/baton-auth0:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-auth0/cmd/baton-auth0@main

baton-auth0

baton resources
```

# Data Model

`baton-auth0` will pull down information about the following resources:
- Users

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually
building spreadsheets. We welcome contributions, and ideas, no matter how
small&mdash;our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# Getting the authentication parameters

The authentication parameters can be found in the Auth0 dashboard under the "Applications" section.

If you don't already have an application, you can create one by clicking the "Create Application" button. Select "Machine to Machine Applications" and give it a name.

Once you have an application, you can find the "Client ID" and "Client Secret" under the "Settings" section. The application should connect to the management API of the domain.

The permissions needed are:
- Read Users
- Read Grants
- Read Organizations
- Read Organization Members
- Read Roles
- Read Role Members


# `baton-auth0` Command Line Usage

```
baton-auth0

Usage:
  baton-auth0 [flags]
  baton-auth0 [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --auth0-base-url string        required: Base URL of the API ($BATON_AUTH0_BASE_URL)
      --auth0-client-id string       required: App client ID ($BATON_AUTH0_CLIENT_ID)
      --auth0-client-secret string   required: App client secret ($BATON_AUTH0_CLIENT_SECRET)
      --client-id string             The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string         The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string                  The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                         help for baton-auth0
      --log-format string            The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string             The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning                 This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync               This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing                    This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                      version for baton-auth0

Use "baton-auth0 [command] --help" for more information about a command.
```
