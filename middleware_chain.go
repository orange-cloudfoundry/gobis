package gobis

import (
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
	rtr.NotFoundHandler = next
	return rtr, nil
}
func (m MiddlewareChainRoutes) Schema() interface{} {
	return []ProxyRoute{}
}
