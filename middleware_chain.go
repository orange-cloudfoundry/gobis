package gobis

import (
	"github.com/gorilla/mux"
	"net/http"
)

type MiddlewareChainRoutes struct {
	routerFactory RouterFactory
}

func NewMiddlewareChainRoutes(routerFactory RouterFactory) *MiddlewareChainRoutes {
	return &MiddlewareChainRoutes{routerFactory}
}

func (m MiddlewareChainRoutes) Handler(route ProxyRoute, params interface{}, next http.Handler) (http.Handler, error) {
	routes := params.([]ProxyRoute)
	rtr, err := m.routerFactory.CreateMuxRouter(routes, route.PathAsStartPath())
	if err != nil {
		return nil, err
	}
	routeMatchAll := rtr.NewRoute().Name(route.Name + "_all")
	routeMatchAll.MatcherFunc(mux.MatcherFunc(func(req *http.Request, rm *mux.RouteMatch) bool {
		return true
	})).Handler(next)
	return rtr, nil
}
func (m MiddlewareChainRoutes) Schema() interface{} {
	return []ProxyRoute{}
}
