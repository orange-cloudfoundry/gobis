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
	"github.com/orange-cloudfoundry/gobis/utils"
	"fmt"
	"github.com/orange-cloudfoundry/gobis/proxy/ctx"
)

type RouterFactory interface {
	CreateMuxRouterRouteService([]models.ProxyRoute, string, *url.URL) (*mux.Router, error)
	CreateMuxRouter([]models.ProxyRoute, string) (*mux.Router, error)
	CreateForwardHandler(models.ProxyRoute) (http.HandlerFunc, error)
	CreateHttpHandler(models.ProxyRoute) (http.Handler, error)
}

type CreateTransportFunc func(models.ProxyRoute) http.RoundTripper

type RouterFactoryService struct {
	CreateTransportFunc CreateTransportFunc
	Middlewares         []RouterMiddlewareFunc
	MuxRouter           *mux.Router
}
type ErrMiddleware string

func (e ErrMiddleware) Error() string {
	return string(e)
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
		entry := log.WithField("route_name", proxyRoute.Name)
		entry.Debug("orange-cloudfoundry/gobis/proxy: Creating handler ...")
		proxyHandler, err := r.CreateForwardHandler(proxyRoute)
		if err != nil {
			return nil, err
		}
		routeMux := rtr.HandleFunc(startPath + proxyRoute.MuxRoute(), proxyHandler).Name(proxyRoute.Name)
		if len(proxyRoute.Methods) > 0 {
			routeMux.Methods(proxyRoute.Methods...)
		}
		entry.Debug("orange-cloudfoundry/gobis/proxy: Finished handler .")
	}
	log.Debug("orange-cloudfoundry/gobis/proxy: Finished creating handlers ...")
	return rtr, nil
}
func (r RouterFactoryService) CreateHttpHandler(proxyRoute models.ProxyRoute) (http.Handler, error) {
	entry := log.WithField("route_name", proxyRoute.Name)
	var err error
	var fwd *forward.Forwarder
	if !proxyRoute.NoBuffer {
		entry.Debug("orange-cloudfoundry/gobis/proxy: Handler for route will use buffer.")
		fwd, err = forward.New(forward.RoundTripper(r.CreateTransportFunc(proxyRoute)))
	} else {
		entry.Debug("orange-cloudfoundry/gobis/proxy: Handler for route will use direct stream.")
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
	entry := log.WithField("route_name", proxyRoute.Name)
	httpHandler, err := r.CreateHttpHandler(proxyRoute)
	if err != nil {
		return nil, err
	}
	forwardHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		restPath := ""
		vars := mux.Vars(req)
		if vars != nil {
			restPath = vars[models.MUX_REST_VAR_KEY]
		}
		ForwardRequest(proxyRoute, req, restPath)
		httpHandler.ServeHTTP(w, req)
	})

	var handler http.Handler
	handler = forwardHandler
	for i := len(r.Middlewares) - 1; i >= 0; i-- {
		middleware := r.Middlewares[i]
		funcName := utils.GetFunctionName(middleware)
		entry.Debugf("orange-cloudfoundry/gobis/proxy: Adding %s middleware ...", funcName)
		handler, err = middleware(proxyRoute, handler)
		if err != nil {
			return nil, ErrMiddleware(fmt.Sprintf("Failed to add middleware %s: %s", funcName, err.Error()))
		}
		entry.Debugf("orange-cloudfoundry/gobis/proxy: Finished adding %s middleware.", funcName)

	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handler.ServeHTTP(w, req)
	}), nil
}

func ForwardRequest(proxyRoute models.ProxyRoute, req *http.Request, restPath string) {
	dirtyHeaders := ctx.GetDirtyHeaders(req)
	for header, oldValue := range dirtyHeaders {
		if oldValue == "" {
			req.Header.Del(header)
			continue
		}
		req.Header.Set(header, oldValue)
	}
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