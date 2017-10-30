# Gobis [![Build Status](https://travis-ci.org/orange-cloudfoundry/gobis.svg?branch=master)](https://travis-ci.org/orange-cloudfoundry/gobis) [![GoDoc](https://godoc.org/github.com/orange-cloudfoundry/gobis?status.svg)](https://godoc.org/github.com/orange-cloudfoundry/gobis)

Gobis is a lightweight API Gateway written in go which can be used programmatically or as a standalone server.

It's largely inspired by [Netflix/zuul](https://github.com/Netflix/zuul).

## Summary

- [installation](#installation)
- [Usage](#usage)
  - [Headers sent by gobis to reversed app](headers-sent-by-gobis-to-reversed-app)
  - [Example with your own router and middlewares](#example-with-your-own-router-and-middlewares)
- [Middlewares](#middlewares)
  - [Create your middleware](#create-your-middleware)
- [Available middlewares](https://github.com/orange-cloudfoundry/gobis-middlewares)
  - [basic2token](https://github.com/orange-cloudfoundry/gobis-middlewares#basic2token): Give the ability to connect an user over basic auth, retrieve a token from an oauth2 server with user information and forward the request with this token.
  - [basic auth](https://github.com/orange-cloudfoundry/gobis-middlewares#basic-auth)
  - [casbin](https://github.com/orange-cloudfoundry/gobis-middlewares#casbin): An authorization library that supports access control models like ACL, RBAC, ABAC
  - [circuit breaker](https://github.com/orange-cloudfoundry/gobis-middlewares#circuit-breaker)
  - [conn limit](https://github.com/orange-cloudfoundry/gobis-middlewares#conn-limit)
  - [cors](https://github.com/orange-cloudfoundry/gobis-middlewares#cors)
  - [oauth2](https://github.com/orange-cloudfoundry/gobis-middlewares#oauth2)
  - [ldap](https://github.com/orange-cloudfoundry/gobis-middlewares#ldap)
  - [rate limit](https://github.com/orange-cloudfoundry/gobis-middlewares#rate-limit)
  - [trace](https://github.com/orange-cloudfoundry/gobis-middlewares#trace)
  - and more see: https://github.com/orange-cloudfoundry/gobis-middlewares
- [Running a standalone server](#running-a-standalone-server)
- [FAQ](#faq)

## Installation

```
go get github/orange-cloudfoundry/gobis
```

## Usage

Gobis provide an handler to make it useable on your server.

You can found found `gobis.ProxyRoute` options in the godoc: https://godoc.org/github.com/orange-cloudfoundry/gobis#ProxyRoute

**Example**:

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis"
        "github.com/orange-cloudfoundry/gobis-middlewares/cors"
        log "github.com/sirupsen/logrus"
        "net/http"
)
func main(){
        configHandler := gobis.DefaultHandlerConfig{
                Routes: []gobis.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                        MiddlewareParams: gobis.InterfaceToMap(cors.CorsConfig{
                                Cors: &cors.CorsOptions{
                                        AllowedOrigins: []string{"http://localhost"},
                                },
                        }),
                    },
                },
        }
        log.SetLevel(log.DebugLevel) // set verbosity to debug for logs
        gobisHandler, err := gobis.NewDefaultHandler(configHandler)
        if err != nil {
                panic(err)
        }
        err = http.ListenAndServe(":8080", gobisHandler)
        if err != nil {
                panic(err)
        }
}
```

You can see doc [DefaultHandlerConfig](https://godoc.org/github.com/orange-cloudfoundry/gobis#DefaultHandlerConfig) to know more about possible parameters.

You can also see doc [ProxyRoute](https://godoc.org/github.com/orange-cloudfoundry/gobis#ProxyRoute) to see available options for routes.

### Headers sent by gobis to reversed app

Gobis will send some headers to the app when the request is forwarded:

- **X-Gobis-Forward**: This is a dummy header to say to the app that the requested was forwarded by gobis.
- **X-Gobis-Username**: User name of a logged user set by a middleware.
- **X-Gobis-Groups**: User's groups of a logged user set by a middleware.

### Example with your own router and middlewares

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis"
        "github.com/orange-cloudfoundry/gobis-middlewares/cors"
        "github.com/gorilla/mux"
        "net/http"
)
func main(){
        configHandler := gobis.DefaultHandlerConfig{
                Routes: []gobis.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                    },
                },
        }
        rtr := mux.NewRouter()
        gobisHandler, err := gobis.NewDefaultHandler(
                    configHandler,
                    gobis.NewRouterFactoryWithMuxRouter(func()*mux.Router{
                        return rtr
                    }, cors.NewCors()),
                )
        if err != nil {
                panic(err)
        }
        rtr.HandleFunc("/gobis/{d:.*}", gobisHandler.ServeHTTP)
        err = http.ListenAndServe(":8080", rtr)
        if err != nil {
                panic(err)
        }
}
```

## Middlewares

Gobis permit to add middlewares on handler to be able to enhance your upstream url, for example:
- add basic auth security
- add monitoring
- add cors headers
- ...

### Create your middleware

You can see example from [cors middleware](https://github.com/orange-cloudfoundry/gobis-middlewares/blob/master/cors.go).

To use it simply add it to your `RouterFactory`.

Here an example

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis"
        log "github.com/sirupsen/logrus"
        "net/http"
)
type TraceConfig struct{
      EnableTrace bool  `mapstructure:"enable_trace" json:"enable_trace" yaml:"enable_trace"`
}
type traceMiddleware struct {}
func (traceMiddleware) Handler(proxyRoute gobis.ProxyRoute, params interface{}, next http.Handler) (http.Handler, error) {
        // Params has been decode route middleware params, this decoded agains schema you gave in schema function
        traceConfig := params.(TraceConfig)
        if !traceConfig.EnableTrace {
            return next, nil
        }
        return TraceHandler(next), nil
}
// Schema function is required in order to gobis to decode params from route and sent it back to handler function through `params`
// It use https://github.com/mitchellh/mapstructure when decode to inject in handler
func (traceMiddleware) Schema() interface{} {
        return TraceConfig{}
}
func TraceHandler(h http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                groups := gobis.Groups(r) // retrieve current user groups set by other middlewares with gobis.AddGroups(r, "mygroup1", "mygroup2")
                user := gobis.Username(r) // retrieve current user name set by other middlewares with gobis.SetUsername(r, "username")
                path := gobis.Path(r) // retrieve the path which will be passed to upstream (wihtout trailling path name on your route)
                routeName := gobis.RouteName(r) // retrieve the current route name which use this handler
                log.Info("Url received: "+ r.URL.String())
                h.ServeHTTP(w, r)
        })
}
func main(){
        configHandler := gobis.DefaultHandlerConfig{
                StartPath: "/gobis",
                Routes: []gobis.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                    },
                },
        }
        log.SetLevel(log.DebugLevel) // set verbosity to debug for logs
        gobisHandler, err := gobis.NewDefaultHandler(
                    configHandler,
                    gobis.NewRouterFactory(&traceMiddleware{}),
                )
        if err != nil {
                panic(err)
        }
        err = http.ListenAndServe(":8080", gobisHandler)
        if err != nil {
                panic(err)
        }
}
```

## Available middlewares

Middlewares are located on repo https://github.com/orange-cloudfoundry/gobis-middlewares

## Running a standalone server

You can run a prepared gobis server with all default middlewares in one command line, see repo https://github.com/orange-cloudfoundry/gobis-server .

This server can be ran on cloud like Kubernetes, Cloud Foundry or Heroku.

## FAQ

### Why this name ?

Gobis is inspired by [zuul](https://github.com/Netflix/zuul) which also a kind of [dinosaur](https://www.wikiwand.com/en/Zuul) 
which come from the family of [Ankylosauridae](https://www.wikiwand.com/en/Ankylosauridae), the [gobis(aurus)](https://www.wikiwand.com/en/Gobisaurus) come also from this family.
