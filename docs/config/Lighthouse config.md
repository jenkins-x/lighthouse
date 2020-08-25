# Lighthouse config

- [Config](#Config)
- [Cookie](#Cookie)
- [GithubOAuthConfig](#GithubOAuthConfig)
- [JobConfig](#JobConfig)
- [ProwConfig](#ProwConfig)


## Config

Config is a read-only snapshot of the config.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [JobConfig](#JobConfig) | Yes |  |
|  |  | [ProwConfig](#ProwConfig) | Yes |  |

## Cookie

Cookie holds the secret returned from github that authenticates the user who authorized this app.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Secret | `secret` | string | No |  |

## GithubOAuthConfig

GithubOAuthConfig is a config for requesting users access tokens from Github API. It also has<br />a Cookie Store that retains user credentials deriving from Github API.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| ClientID | `client_id` | string | Yes |  |
| ClientSecret | `client_secret` | string | Yes |  |
| RedirectURL | `redirect_url` | string | Yes |  |
| Scopes | `scopes` | []string | No |  |
| FinalRedirectURL | `final_redirect_url` | string | Yes |  |
| CookieStore | `-` | *sessions.CookieStore | No |  |

## JobConfig

JobConfig is a type alias for job.Config



## ProwConfig

ProwConfig is a type alias for lighthouse.Config




