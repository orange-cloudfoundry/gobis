package proxy

import (
	"github.com/orange-cloudfoundry/gobis/models"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/vulcand/oxy/forward"
	log "github.com/sirupsen/logrus"
	"net/url"
	"github.com/vulcand/oxy/buffer"
	"strings"
)

type RouterFactory interface {
	CreateMuxRouterRouteService([]models.ProxyRoute, string, *url.URL) (*mux.Router, error)
	CreateMuxRouter([]models.ProxyRoute, string) (*mux.Router, error)
	CreateForwardHandler(models.ProxyRoute) (http.HandlerFunc, error)
	CreateHttpHandler(models.ProxyRoute) (http.Handler, error)
}

type CreateTransportFunc func(models.ProxyRoute) http.RoundTripper
type RouterMiddlewareFunc func(models.ProxyRoute, http.Handler) http.Handler

type RouterFactoryService struct {
	CreateTransportFunc CreateTransportFunc
	Middlewares         []RouterMiddlewareFunc
	MuxRouter           *mux.Router
}

func NewRouterFactory(middlewares ...RouterMiddlewareFunc) RouterFactory {
	return NewRouterFactoryWithMuxRouter(mux.NewRouter(), middlewares...)
}
func NewRouterFactoryWithMuxRouter(muxRouter *mux.Router, middlewares ...RouterMiddlewareFunc) RouterFactory {
	return &RouterFactoryService{
		CreateTransportFunc: func(proxyRoute models.ProxyRoute) http.RoundTripper {
			return NewRouteTransport(proxyRoute)
		},
		Middlewares: middlewares,
		MuxRouter: muxRouter,
	}
}

func (r RouterFactoryService) CreateMuxRouterRouteService(proxyRoutes []models.ProxyRoute, startPath string, forwardedUrl *url.URL) (*mux.Router, error) {
	rtr, err := r.CreateMuxRouter(proxyRoutes, startPath)
	if err != nil {
		return nil, err
	}
	// forward everything which not matching a proxified route to original app
	originalAppHandler, err := r.CreateForwardHandler(models.ProxyRoute{
		Url: forwardedUrl.String(),
		RemoveProxyHeaders: true,
		InsecureSkipVerify: true,
		NoProxy: true,
	})
	if err != nil {
		return nil, err
	}
	rtr.HandleFunc(forwardedUrl.Path, originalAppHandler)
	return rtr, nil
}
func (r RouterFactoryService) CreateMuxRouter(proxyRoutes []models.ProxyRoute, startPath string) (*mux.Router, error) {
	log.Debug("github.com/orange-cloudfoundry/proxy: Creating handlers ...")
	startPath = strings.TrimSuffix(startPath, "/")
	rtr := r.MuxRouter
	for _, proxyRoute := range proxyRoutes {
		log.Debugf("orange-cloudfoundry/gobis/proxy: Creating handler for route '%s' ...", proxyRoute.Name)
		proxyHandler, err := r.CreateForwardHandler(proxyRoute)
		if err != nil {
			return nil, err
		}
		routeMux := rtr.HandleFunc(startPath + proxyRoute.MuxRoute(), proxyHandler)
		if len(proxyRoute.Methods) > 0 {
			routeMux.Methods(proxyRoute.Methods...)
		}
		log.Debugf("orange-cloudfoundry/gobis/proxy: Finished handler for route '%s' ...", proxyRoute.Name)
	}
	log.Debug("orange-cloudfoundry/gobis/proxy: Finished creating handlers ...")
	return rtr, nil
}
func (r RouterFactoryService) CreateHttpHandler(proxyRoute models.ProxyRoute) (http.Handler, error) {
	var err error
	var fwd *forward.Forwarder
	if !proxyRoute.NoBuffer {
		log.Debugf("orange-cloudfoundry/gobis/proxy: Handler for route '%s' will use buffer.", proxyRoute.Name)
		fwd, err = forward.New(forward.RoundTripper(r.CreateTransportFunc(proxyRoute)))
	} else {
		log.Debugf("orange-cloudfoundry/gobis/proxy: Handler for route '%s' will use direct stream.", proxyRoute.Name)
		fwd, err = forward.New(forward.RoundTripper(r.CreateTransportFunc(proxyRoute)), forward.Stream(true))
	}
	if err != nil {
		return nil, err
	}
	if proxyRoute.NoBuffer {
		return fwd, nil
	}
	return buffer.New(fwd, buffer.Retry(`IsNetworkError() && Attempts() < 2`))
}

func (r RouterFactoryService) CreateForwardHandler(proxyRoute models.ProxyRoute) (http.HandlerFunc, error) {
	handler, err := r.CreateHttpHandler(proxyRoute)
	if err != nil {
		return nil, err
	}
	for _, middleware := range r.Middlewares {
		handler = middleware(proxyRoute, handler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		restPath := ""
		vars := mux.Vars(req)
		if vars != nil {
			restPath = vars[models.MUX_REST_VAR_KEY]
		}
		ForwardRequest(proxyRoute, req, restPath)
		handler.ServeHTTP(w, req)
	}), nil
}

func ForwardRequest(proxyRoute models.ProxyRoute, req *http.Request, restPath string) {
	fwdUrl, _ := url.Parse(proxyRoute.Url)
	req.URL.Host = fwdUrl.Host

	req.URL.Scheme = fwdUrl.Scheme
	finalPath := fwdUrl.Path + restPath

	finalPath = strings.TrimSuffix(finalPath, "/")
	req.URL.Path = finalPath
	reqValues := req.URL.Query()
	for key, values := range fwdUrl.Query() {
		for _, value := range values {
			reqValues.Add(key, value)
		}
	}
	req.URL.RawQuery = reqValues.Encode()
	if fwdUrl.User != nil && fwdUrl.User.Username() != "" {
		password, _ := fwdUrl.User.Password()
		req.SetBasicAuth(fwdUrl.User.Username(), password)
	}
	req.RequestURI = req.URL.RequestURI()
}