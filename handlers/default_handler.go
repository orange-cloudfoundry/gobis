package handlers

import (
	"net/http"
	"github.com/orange-cloudfoundry/gobis/models"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/orange-cloudfoundry/gobis/proxy"
	"net/url"
	"github.com/gorilla/mux"
)

type DefaultHandlerConfig struct {
	// Host where server should listen (default to 127.0.0.1)
	Host             string `json:"host" yaml:"host"`
	// Port where server should listen
	Port             int `json:"port" yaml:"port"`
	// List of routes
	Routes           []models.ProxyRoute `json:"routes" yaml:"routes"`
	// Set the path where all path from route should start (e.g.: if set to `/root` request for the next route will be localhost/root/app)
	StartPath        string `json:"start_path" yaml:"start_path"`
	// Forward all request which doesn't match route to this url
	ForwardedUrl     *url.URL `json:"-" yaml:"-"`
	// List of headers which cannot be removed by `sensitive_headers`
	ProtectedHeaders []string `json:"protected_headers" yaml:"protected_headers"`
}
type DefaultHandler struct {
	port      int
	host      string
	muxRouter *mux.Router
}

func NewDefaultHandler(config DefaultHandlerConfig) (GobisHandler, error) {
	return NewDefaultHandlerWithRouterFactory(config, proxy.NewRouterFactory())
}
func NewDefaultHandlerWithRouterFactory(config DefaultHandlerConfig, routerFactory proxy.RouterFactory) (GobisHandler, error) {
	proxy.SetProtectedHeaders(config.ProtectedHeaders)
	muxRouter, err := generateMuxRouter(config, routerFactory)
	if err != nil {
		return nil, err
	}
	return &DefaultHandler{
		port: config.Port,
		host: config.Host,
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
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", host, port)
}
func generateMuxRouter(config DefaultHandlerConfig, routerFactory proxy.RouterFactory) (*mux.Router, error) {
	var err error
	var rtr *mux.Router
	log.Debug("orange-cloudfoundry/gobis/handlers: Creating mux router for routes ...")
	if config.ForwardedUrl == nil {
		rtr, err = routerFactory.CreateMuxRouter(config.Routes, config.StartPath)
	} else {
		rtr, err = routerFactory.CreateMuxRouterRouteService(
			config.Routes,
			config.StartPath,
			config.ForwardedUrl,
		)
	}
	if err != nil {
		return nil, err
	}
	log.Debug("orange-cloudfoundry/gobis/handlers: Finished creating mux router for routes.")
	return rtr, nil
}