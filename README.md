# Gobis [![Build Status](https://travis-ci.org/orange-cloudfoundry/gobis.svg?branch=master)](https://travis-ci.org/orange-cloudfoundry/gobis) [![GoDoc](https://godoc.org/github.com/orange-cloudfoundry/gobis?status.svg)](https://godoc.org/github.com/orange-cloudfoundry/gobis)

Gobis is a lightweight API Gateway written in go which can be used programmatically or as a standalone server.

It's largely inspired by [Netflix/zuul](https://github.com/Netflix/zuul).

## Summary

- [installation](#installation)
- [Running standalone server](#running-standalone-server)
  - [Commands](#commands)
  - [Usage](#usage)
- [Use in your project](#use-in-your-project)
  - [Example with your own router and middlewares](#example-with-your-own-router-and-middlewares)
- [Middlewares](#middlewares)
  - [Create your middleware](#create-your-middleware)
- [Create your middleware](#create-your-middleware)
- [Available middlewares](#available-middlewares)
  - [basic auth](#basic-auth)
  - [casbin](#casbin): An authorization library that supports access control models like ACL, RBAC, ABAC
  - [circuit breaker](#circuit-breaker)
  - [conn limit](#conn-limit)
  - [cors](#cors)
  - [ldap](#ldap)
  - [rate limit](#rate-limit)
  - [trace](#trace)
- [FAQ](#faq)

## Installation

```
go get github/orange-cloudfoundry/gobis
```

If you set your `PATH` with `$GOPATH/bin/` you should have now a `gobis` binary available, this is the standalone server.

## Running standalone server

The standalone server will make available all middlewares you can found in [Available middlewares](#available-middlewares)

**Note**: To enable them in your route see parameters to set on each ones

### Commands

```
NAME:
   gobis - Create a gobis server based on a config file

USAGE:
   gobis [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value     Path to the config file (default: "gobis-config.yml")
   --log-level value, -l value  Log level to use (default: "info")
   --log-json, -j               Write log in json
   --no-color                   Logger will not display colors
   --help, -h                   show help
   --version, -v                print the version
```

### Usage

1. Create a `gobis-config.yml` file where you want to run your server, following this schema:

```yaml
# Host where server should listen (default to 0.0.0.0) 
host: 127.0.0.1 # you can either set 0.0.0.0
# Port where server should listen, if empty it will look for PORT env var and if not found it will be listen on 9080
port: 8080
# List of headers which cannot be removed by `sensitive_headers`
protected_headers: []
# Set the path where all path from route should start (e.g.: if set to `/root` request for the next route will be localhost/root/app)
start_path: ""
routes:
  # Name of your route
- name: myapi
  # Path which gobis handler should listen to
  # You can use globs:
  #   - appending /* will only make requests available in first level of upstream
  #   - appending /** will pass everything to upstream
  path: /app/**
  # Upstream url where all request will be redirected
  # Query parameters can be passed, e.g.: http://localhost?param=1
  # User and password are given as basic auth too (this is not recommended to use it), e.g.: http://user:password@localhost
  url: http://www.mocky.io/v2/595625d22900008702cd71e8
  # List of headers which should not be sent to upstream
  sensitive_headers: []
  # An url to an http proxy to make requests to upstream pass to this
  http_proxy: ""
  # An url to an https proxy to make requests to upstream pass to this
  https_proxy: ""
  # Force to never use proxy even proxy from environment variables
  no_proxy: false
  # By default response from upstream are buffered, it can be issue when sending big files
  # Set to true to stream response
  no_buffer: false
  # Set to true to not send X-Forwarded-* headers to upstream
  remove_proxy_headers: false
  #  An url to an http proxy to make request to upstream pass to this
  methods: []
  # Set to true to not check ssl certificates from upstream (not recommended)
  insecure_skip_verify: false
  # It was made to pass arbitrary params to use it after in gobis middlewares
  # Here you can set cors parameters for cors middleware (see doc relative to middlewares)
  extra_params:
    cors:
      max_age: 12
      allowed_origins:
      - http://localhost
```

2. Run `gobis` in your terminal and server is now started

## Use in your project

Gobis provide an handler to make it useable on your server here an example:

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis/handlers"
        "github.com/orange-cloudfoundry/gobis/models"
        "github.com/orange-cloudfoundry/gobis/utils"
        "github.com/orange-cloudfoundry/gobis/middlewares"
        log "github.com/sirupsen/logrus"
        "net/http"
)
func main(){
        configHandler := handlers.DefaultHandlerConfig{
                Routes: []models.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                        ExtraParams: utils.InterfaceToMap(middlewares.CorsConfig{
                                Cors: &middlewares.CorsOptions{
                                        AllowedOrigins: []string{"http://localhost"},
                                },
                        }),
                    },
                },
        }
        log.SetLevel(log.DebugLevel) // set verbosity to debug for logs
        gobisHandler, err := handlers.NewDefaultHandler(configHandler)
        if err != nil {
                panic(err)
        }
        return http.ListenAndServe(":8080", gobisHandler)
}
```

You can see doc [DefaultHandlerConfig](https://godoc.org/github.com/orange-cloudfoundry/gobis/handlers#DefaultHandlerConfig) to know more about possible parameters.

You can also see doc [ProxyRoute](https://godoc.org/github.com/orange-cloudfoundry/gobis/models#ProxyRoute) to see available options for routes.

### Example with your own router and middlewares

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis/handlers"
        "github.com/orange-cloudfoundry/gobis/proxy"
        "github.com/orange-cloudfoundry/gobis/models"
        "github.com/orange-cloudfoundry/gobis/middlewares"
        "github.com/gorilla/mux"
        "net/http"
)
func main(){
        configHandler := handlers.DefaultHandlerConfig{
                Routes: []models.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                    },
                },
        }
        rtr := mux.NewRouter()
        gobisHandler, err := handlers.NewDefaultHandlerWithRouterFactory(
                    configHandler,
                    proxy.NewRouterFactoryWithMuxRouter(rtr, middlewares.Cors),
                )
        if err != nil {
                panic(err)
        }
        rtr.HandleFunc("/gobis/{d:.*}", gobisHandler)
        return http.ListenAndServe(":8080", rtr)
}
```

## Middlewares

Gobis permit to add middlewares on handler to be able to enhance your upstream url, for example:
- add basic auth security
- add monitoring
- add cors headers
- ...

### Create your middleware

You can see example from [cors middleware](/middlewares/cors.go).

To use it simply add it to your `RouterFactory`.

Here an example

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis/handlers"
        "github.com/orange-cloudfoundry/gobis/proxy"
        "github.com/orange-cloudfoundry/gobis/models"
        log "github.com/sirupsen/logrus"
        "github.com/mitchellh/mapstructure"
        "net/http"
)
type TraceConfig struct{
      EnableTrace string  `mapstructure:"enable_trace" json:"enable_trace" yaml:"enable_trace"`
}
func traceMiddleware(proxyRoute models.ProxyRoute, parentHandler http.Handler) (http.Handler, error) {
        var traceConfig TraceConfig
        mapstructure.Decode(proxyRoute.ExtraParams, &traceConfig)
        if !traceConfig.EnableTrace {
            return parentHandler, nil
        }
        return TraceHandler(parentHandler), nil
}

func TraceHandler(h http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                log.Info("Url received: "+ r.URL.String())
                h.ServeHTTP(w, r)
        })
}
func main(){
        configHandler := handlers.DefaultHandlerConfig{
                StartPath: "/gobis",
                Routes: []models.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                    },
                },
        }
        log.SetLevel(log.DebugLevel) // set verbosity to debug for logs
        gobisHandler, err := handlers.NewDefaultHandlerWithRouterFactory(
                    configHandler,
                    proxy.NewRouterFactory(traceMiddleware),
                )
        if err != nil {
                panic(err)
        }
        return http.ListenAndServe(":8080", gobisHandler)
}
```

## Available middlewares

### Basic2Token

Give the ability to connect an user over basic auth, retrieve a token from an oauth2 server with user information and forward the request with this token.

This was made to transparently convert a basic auth authentication to an oauth2 one.

See godoc for [Basic2TokenOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#Basic2TokenOptions) to know more about parameters.

**Note**:
- Your oauth2 server must have the `password` grant type such as [UAA](https://github.com/cloudfoundry/uaa) or [Gitlab in oauth2 provider](https://docs.gitlab.com/ce/api/oauth2.html#resource-owner-password-credentials)

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.Basic2TokenConfig{
                        Ldap: &middlewares.Basic2TokenOptions{
                                Enable: true,
                                AccessTokenUri: "https://my.uaa.local/oauth/token",
                                ClientId: "cf",
                                ClientSecret: "",
                                ParamsAsJson: false,
                                UseRouteTransport: true,
                                InsecureSkipVerify: true,
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  basic2token:
    enable: true
    access_token_uri: https://my.uaa.local/oauth/token
    client_id: cf
    client_secret: ~
    params_as_json: false
    use_route_transport: false
    insecure_skip_verify: true
```

#### Tips

- If key `scope` is found in the response of the oauth2 server, thoses scopes will be loaded as groups and others middlewares will
 be able to find groups for the current user by using [context groups](https://godoc.org/github.com/orange-cloudfoundry/gobis/proxy/ctx#Groups)
- Logged user is accessible by other middleware through [context username](https://godoc.org/github.com/orange-cloudfoundry/gobis/proxy/ctx#Username)

### Basic auth

Add basic auth to upstream

See godoc for [BasicAuthOption](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#BasicAuthOption) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.BasicAuthConfig{
                        BasicAuth: &middlewares.BasicAuthOptions{
                                {
                                        User: "user",
                                        Password: "$2y$12$AHKssZrkmcG2pmom.rvy2OMsV8HpMHHcRIEY158LgZIkrA0BFvFQq", // equal password
                                        Crypted: true, // hashed by bcrypt, you can use https://github.com/gibsjose/bcrypt-hash command to crypt a password
                                },
                                {
                                        User: "user2",
                                        Password: "mypassword",
                                        Groups: []string{"admin"}
                                },
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  basic_auth:
  - user: user
    password: $2y$12$AHKssZrkmcG2pmom.rvy2OMsV8HpMHHcRIEY158LgZIkrA0BFvFQq # equal password
    crypted: true # hashed by bcrypt, you can use https://github.com/gibsjose/bcrypt-hash command to crypt a password
  - user: user2
    password: mypassword # equal password
    groups: [admin]
```

#### Tips

By setting groups it will allow others middleware to find groups for the current user by using [context groups](https://godoc.org/github.com/orange-cloudfoundry/gobis/proxy/ctx#Groups)

### Casbin

[casbin](https://github.com/casbin/casbin) is an authorization library that supports access control models like ACL, RBAC, ABAC.

This middleware allow you to add access control over your apo

See godoc for [CasbinOption](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares/casbin#CasbinOption) to know more about parameters.

#### Use programmatically

```go
import "github.com/orange-cloudfoundry/gobis/middlewares/casbin"

configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(casbin.CasbinConfig{
                        CircuitBreaker: &casbin.CasbinOption{
                                Enable: true,
                                Policies: []casbin.CasbinPolicy{
                                        Type: "p",
                                        Sub: "usernameOrGroupName",
                                        Obj: "/mysubpath/*"
                                        Act: "*",
                                },
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  casbin:
    enable: true
    policies:
    - {type: p, sub: usernameOrGroupName, obj: /mysubpath/*, act: *}
```

#### Tips

- It will load as role policies all groups found by using [context groups](https://godoc.org/github.com/orange-cloudfoundry/gobis/proxy/ctx#Groups)
this allow you, if you use ldap middleware, to pass a group name found as a `sub` (e.g.: `sub: myUserGroupName`)
- It will also load all policies found in context key `casbin.PolicyContextKey` this allow other middleware to add their own policies

### Circuit breaker

Hystrix-style circuit breaker

See godoc for [CircuitBreakerOption](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#CircuitBreakerOption) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.CircuitBreakerConfig{
                        CircuitBreaker: &middlewares.CircuitBreakerOptions{
                                Enable: true,
                                Expression: "NetworkErrorRatio() < 0.5",
                                FallbackUrl: "http://my.fallback.com",
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  circuit_breaker:
    enable: true
    expression: NetworkErrorRatio() < 0.5
    fallback_url: http://my.fallback.com
```


### Conn limit

Limit number of simultaneous connection

See godoc for [ConnLimitOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#ConnLimitOptions) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.ConnLimitConfig{
                        ConnLimit: &middlewares.ConnLimitOptions{
                                Enable: true,
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  conn_limit:
    enable: true
```

### Cors

Add cors headers to response

See godoc for [CorsOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#CorsOptions) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.CorsConfig{
                        Cors: &middlewares.CorsOptions{
                                AllowedOrigins: []string{"http://localhost"},
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  cors:
    max_age: 12
    allowed_origins:
    - http://localhost
```

### Ldap

Add basic authentiation based on ldap to upstream

See godoc for [LdapOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#LdapOptions) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.LdapConfig{
                        Ldap: &middlewares.LdapOptions{
                                Enable: true,
                                BindDn: "uid=readonly,dc=com",
                                BindPassword: "password",
                                Address: "ldap.example.com:636",
                                InsecureSkipVerify: true,
                                UseSsl: true,
                                SearchBaseDns: "dc=example,dc=com",
                                SearchFilter: "(objectClass=organizationalPerson)&(uid=%s)",
                                GroupSearchBaseDns: "ou=Group,dc=example,dc=com",
                                GroupSearchFilter: "(&(objectClass=posixGroup)(memberUid=%s))",
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  ldap:
    enable: true
    bind_dn: uid=readonly,dc=com
    bind_password: password
    address: ldap.example.com:636
    insecure_skip_verify: true
    use_ssl: true
    search_base_dns: dc=example,dc=com
    search_filter: (objectClass=organizationalPerson)&(uid=%s)
    group_search_base_dns: ou=Group,dc=example,dc=com
    group_search_filter: (&(objectClass=posixGroup)(memberUid=%s))
```

#### Tips

If `GroupSearchBaseDns` and `GroupSearchFilter` params are set the middleware will pass in context 
the list of group accessible by other middlewares by using [context groups](https://godoc.org/github.com/orange-cloudfoundry/gobis/proxy/ctx#Groups)

### Rate limit

Limit number of request in period of time

See godoc for [RateLimitOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#RateLimitOptions) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.RateLimitConfig{
                        RateLimit: &middlewares.RateLimitOptions{
                                Enable: true,
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  rate_limit:
    enable: true
```

### Trace

Structured request and response logger

See godoc for [TraceOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#TraceOptions) to know more about parameters.

#### Use programmatically

```go
configHandler := handlers.DefaultHandlerConfig{
        Routes: []models.ProxyRoute{
            {
                Name: "myapi",
                Path: "/app/**",
                Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                ExtraParams: utils.InterfaceToMap(middlewares.TraceConfig{
                        Trace: &middlewares.TraceOptions{
                                Enable: true,
                        },
                }),
            },
        },
}
gobisHandler, err := handlers.NewDefaultHandler(configHandler)
// create your server
```

#### Use in config file

```yaml
extra_params:
  trace:
    enable: true
```

## FAQ

### Why this name ?

Gobis is inspired by [zuul](https://github.com/Netflix/zuul) which also a kind of [dinosaur](https://www.wikiwand.com/en/Zuul) 
which come from the family of [Ankylosauridae](https://www.wikiwand.com/en/Ankylosauridae), the [gobis(aurus)](https://www.wikiwand.com/en/Gobisaurus) come also from this family.
