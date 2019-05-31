package gobis

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/vulcand/oxy/buffer"
	"github.com/vulcand/oxy/forward"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
)

const (
	GobisHeaderName = "X-Gobis-Forward"
	XGobisUsername  = "X-Gobis-Username"
	XGobisGroups    = "X-Gobis-Groups"
)

type JsonError struct {
	Status    int    `json:"status"`
	Title     string `json:"title"`
	Details   string `json:"details"`
	RouteName string `json:"route_name"`
}

type RouterFactory interface {
	CreateMuxRouter([]ProxyRoute, string) (*mux.Router, error)
	CreateForwardHandler(ProxyRoute) (http.HandlerFunc, error)
	CreateReverseHandler(ProxyRoute) (http.Handler, error)
}

type CreateTransportFunc func(ProxyRoute) http.RoundTripper

type RouterFactoryService struct {
	CreateTransportFunc CreateTransportFunc
	MiddlewareHandlers  []MiddlewareHandler
	muxRouterFunc       func() *mux.Router
	middlewareChain     *MiddlewareChainRoutes
}
type ErrMiddleware string

func (e ErrMiddleware) Error() string {
	return string(e)
}

func NewRouterFactory(middlewareHandlers ...MiddlewareHandler) RouterFactory {
	return NewRouterFactoryWithMuxRouter(func() *mux.Router {
		return mux.NewRouter()
	}, middlewareHandlers...)
}

func NewRouterFactoryWithMuxRouter(muxRouterOption func() *mux.Router, middlewares ...MiddlewareHandler) RouterFactory {
	factory := &RouterFactoryService{
		CreateTransportFunc: func(proxyRoute ProxyRoute) http.RoundTripper {
			return NewRouteTransport(proxyRoute)
		},
		MiddlewareHandlers: middlewares,
		muxRouterFunc:      muxRouterOption,
	}
	factory.middlewareChain = NewMiddlewareChainRoutes(factory)
	return factory
}

func (r RouterFactoryService) CreateMuxRouter(proxyRoutes []ProxyRoute, startPath string) (*mux.Router, error) {
	log.Debug("github.com/orange-cloudfoundry/proxy: Creating handlers ...")
	startPath = strings.TrimSuffix(startPath, "/")
	parentRtr := r.muxRouterFunc()
	var rtr *mux.Router
	if startPath != "" {
		rtr = parentRtr.PathPrefix(startPath).Subrouter()
	} else {
		rtr = parentRtr
	}
	for _, proxyRoute := range proxyRoutes {
		entry := log.WithField("route_name", proxyRoute.Name)
		entry.Debug("orange-cloudfoundry/gobis/proxy: Creating handler ...")
		proxyHandler, err := r.CreateForwardHandler(proxyRoute)
		if err != nil {
			return nil, err
		}

		rtr.NewRoute().
			Name(proxyRoute.Name).
			MatcherFunc(r.routeMatch(proxyRoute, startPath)).
			Handler(proxyHandler)
		entry.Debug("orange-cloudfoundry/gobis/proxy: Finished handler .")
	}
	log.Debug("orange-cloudfoundry/gobis/proxy: Finished creating handlers ...")
	return parentRtr, nil
}

func (r RouterFactoryService) CreateReverseHandler(proxyRoute ProxyRoute) (http.Handler, error) {
	entry := log.WithField("route_name", proxyRoute.Name)
	if proxyRoute.ForwardHandler != nil {
		entry.Debug("orange-cloudfoundry/gobis/proxy: Handler for routes will use forward handler provided.")
		return proxyRoute.ForwardHandler, nil
	}
	var err error
	var fwd *forward.Forwarder
	if !proxyRoute.NoBuffer {
		entry.Debug("orange-cloudfoundry/gobis/proxy: Handler for routes will use buffer.")
		fwd, err = forward.New(forward.RoundTripper(r.CreateTransportFunc(proxyRoute)))
	} else {
		entry.Debug("orange-cloudfoundry/gobis/proxy: Handler for routes will use direct stream.")
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

func (r RouterFactoryService) routeMatch(proxyRoute ProxyRoute, startPath string) mux.MatcherFunc {
	return mux.MatcherFunc(func(req *http.Request, rm *mux.RouteMatch) bool {
		if len(proxyRoute.Methods) > 0 && !funk.ContainsString(proxyRoute.Methods, req.Method) {
			return false
		}
		path := proxyRoute.RequestPath(req)
		if startPath != "" {
			path = strings.TrimPrefix(path, startPath)
		}
		matcher := proxyRoute.RouteMatcher()
		if !matcher.MatchString(path) {
			return false
		}
		sub := matcher.FindStringSubmatch(path)
		finalPath := ""
		if len(sub) >= 2 {
			finalPath = sub[1]
		}
		SetPath(req, finalPath)
		upstreamUrl := proxyRoute.UpstreamUrl(req)
		if proxyRoute.ForwardedHeader != "" {
			req.URL.RawQuery = upstreamUrl.RawQuery
		}
		if proxyRoute.Url == "" || proxyRoute.ForwardedHeader == "" {
			return true
		}
		origUpstreamUrl, _ := url.Parse(proxyRoute.Url)
		if origUpstreamUrl.Host != upstreamUrl.Host {
			return false
		}
		if origUpstreamUrl.Path == "" || origUpstreamUrl.Path == "/" {
			return true
		}
		origPathMatcher := NewPathMatcher(origUpstreamUrl.Path).pathMatcher
		return origPathMatcher.MatchString(path)
	})
}

func (r RouterFactoryService) CreateForwardHandler(proxyRoute ProxyRoute) (http.HandlerFunc, error) {
	httpHandler, err := r.CreateReverseHandler(proxyRoute)
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

	for i := len(r.MiddlewareHandlers) - 1; i >= 0; i-- {
		middleware := r.MiddlewareHandlers[i]
		params := paramsToSchema(proxyRoute.MiddlewareParams, middleware.Schema())
		handler, err = middlewareHandlerToHandler(middleware, proxyRoute, params, handler)
		if err != nil {
			return nil, err
		}
	}

	if len(proxyRoute.Routes) > 0 {
		handler, err = middlewareHandlerToHandler(r.middlewareChain, proxyRoute, proxyRoute.Routes, handler)
		if err != nil {
			return nil, err
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Header.Set(GobisHeaderName, "true")
		w.Header().Set(GobisHeaderName, "true")
		if proxyRoute.OptionsPassthrough &&
			req.Method == "OPTIONS" &&
			(req.Header.Get("Access-Control-Request-Method") != "" || req.Header.Get("Access-Control-Request-Headers") != "") {
			forwardHandler.ServeHTTP(w, req)
			return
		}
		if proxyRoute.HostsPassthrough.Match(req.URL.Host) {
			forwardHandler.ServeHTTP(w, req)
			return
		}
		setRouteName(req, proxyRoute.Name)
		defer panicRecover(proxyRoute, w)
		handler.ServeHTTP(w, req)
	}), nil
}

func middlewareHandlerToHandler(middleware MiddlewareHandler, proxyRoute ProxyRoute, params interface{}, next http.Handler) (http.Handler, error) {
	entry := log.WithField("route_name", proxyRoute.Name)
	funcName := GetMiddlewareName(middleware)
	entry.Debugf("orange-cloudfoundry/gobis/proxy: Adding %s middleware ...", funcName)
	handler, err := middleware.Handler(proxyRoute, params, next)
	if err != nil {
		return nil, ErrMiddleware(fmt.Sprintf("Failed to add middleware %s: %s", funcName, err.Error()))
	}
	entry.Debugf("orange-cloudfoundry/gobis/proxy: Finished adding %s middleware.", funcName)
	return handler, nil
}

func paramsToSchema(params interface{}, schema interface{}) interface{} {
	if params != nil && reflect.TypeOf(params).Kind() != reflect.Map {
		params = InterfaceToMap(params)
	}
	val := reflect.New(reflect.TypeOf(schema))
	err := mapstructure.Decode(params, val.Interface())
	if err != nil {
		panic(err)
	}
	return val.Elem().Interface()
}

func ForwardRequest(proxyRoute ProxyRoute, req *http.Request, restPath string) {
	removeDirtyHeaders(req)
	req.Header.Add(XGobisUsername, Username(req))
	req.Header.Add(XGobisGroups, strings.Join(Groups(req), ","))
	fwdUrl := proxyRoute.UpstreamUrl(req)
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
	w.WriteHeader(http.StatusInternalServerError)
	if proxyRoute.ShowError {
		w.Header().Set("Content-Type", "application/json")
		errMsg := JsonError{
			Status:    http.StatusInternalServerError,
			Title:     http.StatusText(http.StatusInternalServerError),
			Details:   fmt.Sprint(err),
			RouteName: proxyRoute.Name,
		}
		b, _ := json.MarshalIndent(errMsg, "", "\t")
		w.Write([]byte(b))
	}
	entry := log.WithField("route_name", proxyRoute.Name)
	identName, identFile := identifyPanic()
	if identName != "" {
		entry.WithField("func", identName)
	}
	entry.WithField("file", identFile)
	entry.Error(err)
}

func identifyPanic() (string, string) {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}
	filefinal := fmt.Sprintf("%s:%d", file, line)
	if file == "" && name == "" {
		return "", fmt.Sprintf("pc:%x", pc)
	}
	return name, filefinal

}
