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
	config        DefaultHandlerConfig
	routerFactory proxy.RouterFactory
}

func NewDefaultHandler(config DefaultHandlerConfig) GobisHandler {
	proxy.SetProtectedHeaders(config.ProtectedHeaders)
	return &DefaultHandler{
		config: config,
		routerFactory: proxy.NewRouterFactory(),
	}
}
func NewDefaultHandlerWithRouterFactory(config DefaultHandlerConfig, routerFactory proxy.RouterFactory) GobisHandler {
	proxy.SetProtectedHeaders(config.ProtectedHeaders)
	return &DefaultHandler{
		config: config,
		routerFactory: routerFactory,
	}
}
func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	var rtr *mux.Router
	log.Debug("orange-cloudfoundry/gobis/handlers: Creating mux router for routes ...")
	if h.config.ForwardedUrl == nil {
		rtr, err = h.routerFactory.CreateMuxRouter(h.config.Routes, h.config.StartPath)
	} else {
		rtr, err = h.routerFactory.CreateMuxRouterRouteService(
			h.config.Routes,
			h.config.StartPath,
			h.config.ForwardedUrl,
		)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something bad happened with router: " + err.Error()))
		log.Errorf("github.com/orange-cloudfoundry/gobis/handlers: Error when creating router: %s", err.Error())
		return
	}
	log.Debug("orange-cloudfoundry/gobis/handlers: Finished creating mux router for routes.")
	rtr.ServeHTTP(w, req)

}

func (h DefaultHandler) GetServerAddr() string {
	port := h.config.Port
	if port == 0 {
		port = 9080
	}
	host := h.config.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", host, port)
}