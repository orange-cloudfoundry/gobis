# Gobis [![Build Status](https://travis-ci.org/orange-cloudfoundry/gobis.svg?branch=master)](https://travis-ci.org/orange-cloudfoundry/gobis) [![GoDoc](https://godoc.org/github.com/orange-cloudfoundry/gobis?status.svg)](https://godoc.org/github.com/orange-cloudfoundry/gobis)

Gobis is a lightweight API Gateway written in go which can be used programmatically or as a standalone server.

It's largely inspired by [Netflix/zuul](https://github.com/Netflix/zuul).

## Installation

```
go get github/orange-cloudfoundry/gobis
```

If you set your `PATH` with `$GOPATH/bin/` you should have now a `gobis` binary available, this is the standalone server.

## Running standalone server

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
# Host where server should listen (default to 127.0.0.1) 
host: 127.0.0.1 # you can either set 0.0.0.0
# Port where server should listen
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
##
#  extra_params:
#    cors:
#      max_age: 12
#      allowed_origins:
#      - http://localhost
```

2. Run `gobis` and server is now started

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
                        ExtraParams: utils.InterfaceToMap(middlewares.CorsStruct{
                                Cors: &middlewares.CorsOptions{
                                        AllowedOrigins: []string{"http://localhost"},
                                },
                        }),
                    },
                },
        }
        log.SetLevel(log.DebugLevel) // set verbosity to debug for logs
        gobisHandler := handlers.NewDefaultHandler(configHandler)
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
                StartPath: "/gobis",
                Routes: []models.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                    },
                },
        }
        gobisHandler := handlers.NewDefaultHandlerWithRouterFactory(
                    configHandler,
                    proxy.NewRouterFactory(middlewares.Cors),
                )
        rtr := mux.NewRouter()
        rtr.HandleFunc("/gobis/{d:.*}", gobisHandler)
        return http.ListenAndServe(":8080", gobisHandler)
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
func traceMiddleware(proxyRoute models.ProxyRoute, parentHandler http.Handler) http.Handler {
        var traceConfig TraceConfig
        mapstructure.Decode(proxyRoute.ExtraParams, &traceConfig)
        if !traceConfig.EnableTrace {
            return parentHandler
        }
        return TraceHandler(parentHandler)
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
        gobisHandler := handlers.NewDefaultHandlerWithRouterFactory(
                    configHandler,
                    proxy.NewRouterFactory(traceMiddleware),
                )
        return http.ListenAndServe(":8080", gobisHandler)
}
```

## Available middlewares

### Cors (Used by default)

Add cors headers to response

See godoc for [CorsOptions](https://godoc.org/github.com/orange-cloudfoundry/gobis/middlewares#CorsOptions) to know more about parameters.

#### Use programmatically

```go
package main
import (
        "github.com/orange-cloudfoundry/gobis/handlers"
        "github.com/orange-cloudfoundry/gobis/models"
        "github.com/orange-cloudfoundry/gobis/utils"
        "github.com/orange-cloudfoundry/gobis/middlewares"
)
func main(){
        configHandler := handlers.DefaultHandlerConfig{
                Routes: []models.ProxyRoute{
                    {
                        Name: "myapi",
                        Path: "/app/**",
                        Url: "http://www.mocky.io/v2/595625d22900008702cd71e8",
                        ExtraParams: utils.InterfaceToMap(middlewares.CorsStruct{
                                Cors: &middlewares.CorsOptions{
                                        AllowedOrigins: []string{"http://localhost"},
                                },
                        }),
                    },
                },
        }
        gobisHandler := handlers.NewDefaultHandler(configHandler)
        // create your server
}
```

#### Use in config file

```yaml
extra_params:
  cors:
    max_age: 12
    allowed_origins:
    - http://localhost
```

## Why this name ?

Gobis is inspired by [zuul](https://github.com/Netflix/zuul) which also a kind of [dinosaur](https://www.wikiwand.com/en/Zuul) 
which come from the family of [Ankylosauridae](https://www.wikiwand.com/en/Ankylosauridae), the [gobis(aurus)](https://www.wikiwand.com/en/Gobisaurus) come also from this family.
