package gobis

import (
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type DefaultHandlerConfig struct {
	// List of routes
	Routes []ProxyRoute `json:"routes" yaml:"routes"`
	// List of headers which cannot be removed by `sensitive_headers`
	ProtectedHeaders []string `json:"protected_headers" yaml:"protected_headers"`
	// Set the path where all path from routes should start (e.g.: if set to `/root` request for the next routes will be localhost/root/app)
	StartPath string `json:"start_path" yaml:"start_path"`
}

type MiddlewareConfig struct {
	// List of routes
	Routes []ProxyRoute `json:"routes" yaml:"routes"`
	// Set the path where all path from routes should start (e.g.: if set to `/root` request for the next routes will be localhost/root/app)
	StartPath string `json:"start_path" yaml:"start_path"`
}

type DefaultHandler struct {
	port      int
	host      string
	muxRouter *mux.Router
}

func NewHandler(routes []ProxyRoute, middlewareHandlers ...MiddlewareHandler) (GobisHandler, error) {
	return NewDefaultHandler(DefaultHandlerConfig{
		Routes: routes,
	}, middlewareHandlers...)
}

func NewHandlerWithFactory(routes []ProxyRoute, factory RouterFactory) (GobisHandler, error) {
	config := DefaultHandlerConfig{
		Routes: routes,
	}
	SetProtectedHeaders(config.ProtectedHeaders)
	muxRouter, err := generateMuxRouter(config, factory)
	if err != nil {
		return nil, err
	}
	return &DefaultHandler{
		muxRouter: muxRouter,
	}, nil
}

func NewDefaultHandler(config DefaultHandlerConfig, middlewareHandlers ...MiddlewareHandler) (GobisHandler, error) {
	SetProtectedHeaders(config.ProtectedHeaders)
	muxRouter, err := generateMuxRouter(config, NewRouterFactory(middlewareHandlers...))
	if err != nil {
		return nil, err
	}
	return &DefaultHandler{
		muxRouter: muxRouter,
	}, nil
}

func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.muxRouter.ServeHTTP(w, req)
}

func (h DefaultHandler) GetServerAddr() string {
	port := h.port
	if port == 0 {
		port = 9080
	}
	host := h.host
	if host == "" {
		host = "0.0.0.0"
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func generateMuxRouter(config DefaultHandlerConfig, routerFactory RouterFactory) (*mux.Router, error) {
	var err error
	var rtr *mux.Router
	log.Debug("orange-cloudfoundry/gobis/handlers: Creating mux router for routes ...")
	rtr, err = routerFactory.CreateMuxRouter(config.Routes, config.StartPath)
	if err != nil {
		return nil, err
	}
	log.Debug("orange-cloudfoundry/gobis/handlers: Finished creating mux router for routes.")
	return rtr, nil
}

func NewGobisMiddleware(routes []ProxyRoute, middlewareHandlers ...MiddlewareHandler) (func(next http.Handler) http.Handler, error) {
	log.Debug("orange-cloudfoundry/gobis/middleware: Creating mux router for routes ...")
	rtr, err := NewRouterFactory(middlewareHandlers...).CreateMuxRouter(routes, "")
	if err != nil {
		return nil, err
	}

	log.Debug("orange-cloudfoundry/gobis/middleware: Finished creating mux router for routes.")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var match mux.RouteMatch
			if !rtr.Match(req, &match) {
				next.ServeHTTP(w, req)
				return
			}
			rtr.ServeHTTP(w, req)
		})
	}, nil
}
