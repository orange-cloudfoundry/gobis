package gobistest

import (
	"github.com/orange-cloudfoundry/gobis"
	"net/http"
	"strings"
)

type GobisHandlerTest struct {
	routes       []gobis.ProxyRoute
	servers      []*PackServer
	gobisHandler gobis.GobisHandler
}

func generateRoutesAndServers(routes []gobis.ProxyRoute, inSsl bool) ([]gobis.ProxyRoute, []*PackServer) {
	servers := make([]*PackServer, 0)
	finalRoutes := make([]gobis.ProxyRoute, 0)

	for i, route := range routes {
		route.NoBuffer = true
		servers = append(servers, CreateBackendServer(route.Name))
		if len(route.Routes) > 0 {
			subRoutes, subServers := generateRoutesAndServers(route.Routes, inSsl)
			servers = append(servers, subServers...)
			route.Routes = subRoutes
		}
		routeUrl := servers[i].Server.URL
		if inSsl {
			routeUrl = strings.TrimPrefix(routeUrl, "http")
			routeUrl = "https" + routeUrl
		}
		if route.ForwardedHeader == "" {
			route.Url = routeUrl
		}
		finalRoutes = append(finalRoutes, route)

	}
	return finalRoutes, servers
}
func NewSimpleGobisHandlerTest(routes ...gobis.ProxyRoute) *GobisHandlerTest {
	return NewGobisHandlerTest(routes)
}
func NewSimpleGobisHandlerTestInSsl(routes ...gobis.ProxyRoute) *GobisHandlerTest {
	return NewGobisHandlerTestSsl(routes, true)
}
func NewGobisHandlerTest(routes []gobis.ProxyRoute, middlewareHandlers ...gobis.MiddlewareHandler) *GobisHandlerTest {
	return NewGobisHandlerTestSsl(routes, false, middlewareHandlers...)
}
func NewGobisHandlerTestSsl(routes []gobis.ProxyRoute, inSsl bool, middlewareHandlers ...gobis.MiddlewareHandler) *GobisHandlerTest {
	finalRoutes, servers := generateRoutesAndServers(routes, inSsl)

	config := gobis.DefaultHandlerConfig{
		Routes: finalRoutes,
	}
	gobisHandler, err := gobis.NewDefaultHandler(config, middlewareHandlers...)
	if err != nil {
		panic(err)
	}
	return &GobisHandlerTest{
		routes:       finalRoutes,
		servers:      servers,
		gobisHandler: gobisHandler,
	}
}
func (h GobisHandlerTest) ServerFirst() *PackServer {
	return h.servers[0]
}
func (h GobisHandlerTest) Server(route gobis.ProxyRoute) *PackServer {
	return h.ServerByName(route.Name)
}
func (h GobisHandlerTest) ServerByName(name string) *PackServer {
	for _, server := range h.servers {
		if server.Name == name {
			return server
		}
	}
	panic("Can't found server " + name)
}
func (h *GobisHandlerTest) SetBackendHandlerByName(name string, handler http.Handler) {
	server := h.ServerByName(name)
	server.SetHandler(handler)
}
func (h *GobisHandlerTest) SetBackendHandler(route gobis.ProxyRoute, handler http.Handler) {
	h.SetBackendHandlerByName(route.Name, handler)
}
func (h *GobisHandlerTest) SetBackendHandlerFirst(handler http.Handler) {
	server := h.ServerFirst()
	server.SetHandler(handler)
}
func (h GobisHandlerTest) Close() {
	for _, server := range h.servers {
		server.Server.Close()
	}
}
func (h GobisHandlerTest) GetServerAddr() string {
	return h.gobisHandler.GetServerAddr()
}
func (h GobisHandlerTest) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.gobisHandler.ServeHTTP(w, req)
}
