package gobistest

import (
	"github.com/orange-cloudfoundry/gobis"
	"net/http"
	"net/http/httptest"
	"strings"
)

const DEFAULT_HANDLER_RESPONSE = "i'm the backend"

type MiddlewareTest struct {
	rr                 *httptest.ResponseRecorder
	route              gobis.ProxyRoute
	backendHandler     http.Handler
	middlewareHandlers []gobis.MiddlewareHandler
}

func NewSimpleMiddlewareTest(middlewareParams interface{}, middlewareHandlers ...gobis.MiddlewareHandler) *MiddlewareTest {
	midNames := make([]string, len(middlewareHandlers))
	for i, middleware := range middlewareHandlers {
		midNames[i] = gobis.GetMiddlewareName(middleware)
	}
	routeName := "route_" + strings.ToLower(strings.Join(midNames, "_"))
	route := gobis.ProxyRoute{
		Name:             routeName,
		Path:             gobis.NewPathMatcher("/**"),
		ShowError:        true,
		MiddlewareParams: middlewareParams,
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(DEFAULT_HANDLER_RESPONSE))
	})
	return NewMiddlewareTest(route, handler, middlewareHandlers...)
}
func NewMiddlewareTest(route gobis.ProxyRoute, backendHandler http.Handler, middlewareHandlers ...gobis.MiddlewareHandler) *MiddlewareTest {
	return &MiddlewareTest{
		route:              route,
		backendHandler:     backendHandler,
		middlewareHandlers: middlewareHandlers,
	}
}
func (t *MiddlewareTest) Run(req *http.Request) *http.Response {
	gobisHandler := NewGobisHandlerTest([]gobis.ProxyRoute{t.route}, t.middlewareHandlers...)
	defer gobisHandler.Close()
	gobisHandler.SetBackendHandlerFirst(t.backendHandler)

	t.rr = httptest.NewRecorder()

	gobisHandler.ServeHTTP(t.rr, req)

	return t.rr.Result()
}

func (t MiddlewareTest) ResponseRecorder() *httptest.ResponseRecorder {
	return t.rr
}
func (t MiddlewareTest) ResponseWriter() http.ResponseWriter {
	return t.rr
}
func (t *MiddlewareTest) SetMiddlewareParams(middlewareParams interface{}) {
	route := t.route
	route.MiddlewareParams = middlewareParams
	t.route = route
}
func (t *MiddlewareTest) AddMiddlewareParam(rootKey string, key string, value interface{}) {
	if _, ok := t.route.MiddlewareParams.(map[string]interface{}); !ok {
		panic("Middleware params is not a map[string]interface{} type. ")
	}
	rootParams := t.route.MiddlewareParams.(map[string]interface{})
	params := rootParams[rootKey].(map[string]interface{})
	params[key] = value
	rootParams[rootKey] = params
	t.SetMiddlewareParams(rootParams)
}
func (t *MiddlewareTest) AddMiddlewareParamToFirst(key string, value interface{}) {
	if _, ok := t.route.MiddlewareParams.(map[string]interface{}); !ok {
		panic("Middleware params is not a map[string]interface{} type. ")
	}
	rootParams := t.route.MiddlewareParams.(map[string]interface{})
	var rootKey string
	for rootKey = range rootParams {
		break
	}
	t.AddMiddlewareParam(rootKey, key, value)
}
func (t *MiddlewareTest) SetMiddlewares(middlewareHandlers []gobis.MiddlewareHandler) {
	t.middlewareHandlers = middlewareHandlers
}
func (t *MiddlewareTest) AddMiddlewares(middlewareHandlers ...gobis.MiddlewareHandler) {
	t.middlewareHandlers = append(t.middlewareHandlers, middlewareHandlers...)
}
func (t *MiddlewareTest) CleanMiddlewares() {
	t.middlewareHandlers = make([]gobis.MiddlewareHandler, 0)
}
func (t *MiddlewareTest) SetRoute(route gobis.ProxyRoute) {
	t.route = route
}
func (t *MiddlewareTest) SetBackendHandler(handler http.Handler) {
	t.backendHandler = handler
}
func (t MiddlewareTest) CreateRequest(methods ...string) *http.Request {
	return CreateRequest(t.route, methods...)
}
