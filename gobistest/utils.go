package gobistest

import (
	"fmt"
	"github.com/orange-cloudfoundry/gobis"
	"net/http"
	"net/http/httptest"
)

type PackServer struct {
	Handler *ParamHandler
	Server  *httptest.Server
	Name    string
}
type ParamHandler struct {
	Handler http.Handler
}

var OriginUrl string = "http://local.app.com"

func (h ParamHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.Handler != nil {
		h.Handler.ServeHTTP(w, req)
	}
}
func (p *PackServer) SetHandler(handler http.Handler) {
	p.Handler.Handler = handler
}
func CreateAppRequest(proxyRoute gobis.ProxyRoute, methods ...string) *http.Request {
	proxyRoute.LoadParams()
	appPath := proxyRoute.AppPath
	if appPath == "" {
		appPath = "/"
	}
	finalUrl := fmt.Sprintf("%s%s", OriginUrl, appPath)
	method := "GET"
	if len(methods) > 0 {
		method = methods[0]
	}
	req, err := http.NewRequest(method, finalUrl, nil)
	if err != nil {
		panic(err)
	}
	return req
}
func CreateBackendServer(name string) *PackServer {
	paramHandler := &ParamHandler{}
	backendServer := httptest.NewServer(paramHandler)
	return &PackServer{paramHandler, backendServer, name}
}
