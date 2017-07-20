package gobis

import (
	"github.com/gorilla/mux"
	"net/http"
	"github.com/vulcand/oxy/forward"
	log "github.com/sirupsen/logrus"
	"net/url"
	"github.com/vulcand/oxy/buffer"
	"strings"
	"fmt"
	"regexp"
	"encoding/json"
)

const (
	GobisHeaderName = "X-Gobis-Forward"
)

type RouterFactory interface {
	CreateMuxRouterRouteService([]ProxyRoute, string, *url.URL) (*mux.Router, error)
	CreateMuxRouter([]ProxyRoute, string) (*mux.Router, error)
	CreateForwardHandler(ProxyRoute) (http.HandlerFunc, error)
	CreateHttpHandler(ProxyRoute) (http.Handler, error)
}

type CreateTransportFunc func(ProxyRoute) http.RoundTripper

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
		CreateTransportFunc: func(proxyRoute ProxyRoute) http.RoundTripper {
			return NewRouteTransport(proxyRoute)
		},
		Middlewares: middlewares,
		MuxRouter: muxRouter,
	}
}

func (r RouterFactoryService) CreateMuxRouterRouteService(proxyRoutes []ProxyRoute, startPath string, forwardedUrl *url.URL) (*mux.Router, error) {
	rtr, err := r.CreateMuxRouter(proxyRoutes, startPath)
	if err != nil {
		return nil, err
	}
	// forward everything which not matching a proxified route to original app
	originalAppHandler, err := r.CreateForwardHandler(ProxyRoute{
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
func (r RouterFactoryService) CreateMuxRouter(proxyRoutes []ProxyRoute, startPath string) (*mux.Router, error) {
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

		routeMux := rtr.NewRoute().
			Name(proxyRoute.Name).
			MatcherFunc(r.routeMatch(proxyRoute)).
			Handler(proxyHandler)
		if len(proxyRoute.Methods) > 0 {
			routeMux.Methods(proxyRoute.Methods...)
		}
		entry.Debug("orange-cloudfoundry/gobis/proxy: Finished handler .")
	}
	log.Debug("orange-cloudfoundry/gobis/proxy: Finished creating handlers ...")
	return rtr, nil
}
func (r RouterFactoryService) CreateHttpHandler(proxyRoute ProxyRoute) (http.Handler, error) {
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
func (r RouterFactoryService) routeMatch(proxyRoute ProxyRoute) (mux.MatcherFunc) {
	return mux.MatcherFunc(func(req *http.Request, rm *mux.RouteMatch) bool {
		matcher := proxyRoute.RouteMatcher()
		reg := regexp.MustCompile(matcher)
		if !reg.MatchString(req.URL.Path) {
			return false
		}
		sub := reg.FindStringSubmatch(req.URL.Path)
		SetPath(req, sub[1])
		return true
	})
}
func (r RouterFactoryService) CreateForwardHandler(proxyRoute ProxyRoute) (http.HandlerFunc, error) {
	entry := log.WithField("route_name", proxyRoute.Name)
	httpHandler, err := r.CreateHttpHandler(proxyRoute)
	if err != nil {
		return nil, err
	}
	forwardHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Del(GobisHeaderName)
		restPath := Path(req)
		ForwardRequest(proxyRoute, req, restPath)
		httpHandler.ServeHTTP(w, req)
	})

	var handler http.Handler
	handler = forwardHandler
	for i := len(r.Middlewares) - 1; i >= 0; i-- {
		middleware := r.Middlewares[i]
		funcName := GetFunctionName(middleware)
		entry.Debugf("orange-cloudfoundry/gobis/proxy: Adding %s middleware ...", funcName)
		handler, err = middleware(proxyRoute, handler)
		if err != nil {
			return nil, ErrMiddleware(fmt.Sprintf("Failed to add middleware %s: %s", funcName, err.Error()))
		}
		entry.Debugf("orange-cloudfoundry/gobis/proxy: Finished adding %s middleware.", funcName)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		setRouteName(req, proxyRoute.Name)
		req.Header.Set(GobisHeaderName, "true")
		w.Header().Set(GobisHeaderName, "true")
		defer panicRecover(proxyRoute, w)
		handler.ServeHTTP(w, req)
	}), nil
}

func ForwardRequest(proxyRoute ProxyRoute, req *http.Request, restPath string) {
	removeDirtyHeaders(req)
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
func removeDirtyHeaders(req *http.Request) {
	dirtyHeadersPtr := DirtyHeaders(req)
	if dirtyHeadersPtr == nil {
		return
	}
	dirtyHeaders := *dirtyHeadersPtr
	for header, oldValue := range dirtyHeaders {
		if oldValue == "" {
			req.Header.Del(header)
			continue
		}
		req.Header.Set(header, oldValue)
	}
}
func panicRecover(proxyRoute ProxyRoute, w http.ResponseWriter) {
	err := recover()
	if err == nil {
		return
	}
	if proxyRoute.ShowError {
		w.Header().Set("Content-Type", "application/json")
		errMsg := struct {
			Status    int `json:"status"`
			Title     string `json:"title"`
			Details   string `json:"details"`
			RouteName string `json:"route_name"`
		}{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), fmt.Sprint(err), proxyRoute.Name}
		b, _ := json.MarshalIndent(errMsg, "", "\t")
		w.Write([]byte(b))
	}
	w.WriteHeader(http.StatusInternalServerError)
	entry := log.WithField("route_name", proxyRoute.Name)
	entry.Error(err)
}