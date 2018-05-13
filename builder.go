package gobis

import (
	"github.com/satori/go.uuid"
	"net/http"
	"reflect"
)

type Builder struct {
	routes   []*ProxyRoute
	children map[int]*Builder
	parent   *Builder
	index    int
}

func NewProxyRouteBuilder() *Builder {
	return &Builder{
		routes:   make([]*ProxyRoute, 0),
		children: make(map[int]*Builder, 0),
		index:    -1,
	}
}

func (b *Builder) AddRoute(path, url string) *Builder {
	b.routes = append(b.routes, &ProxyRoute{
		Name:             uuid.NewV4().String(),
		Path:             path,
		Url:              url,
		SensitiveHeaders: []string{},
		Methods:          []string{},
		MiddlewareParams: map[string]interface{}{},
	})
	b.index++
	return b
}

func (b *Builder) AddRouteHandler(path string, forwardHandler http.Handler) *Builder {
	b.routes = append(b.routes, &ProxyRoute{
		Name:             uuid.NewV4().String(),
		Path:             path,
		ForwardHandler:   forwardHandler,
		SensitiveHeaders: []string{},
		Methods:          []string{},
		MiddlewareParams: map[string]interface{}{},
	})
	b.index++
	return b
}

func (b *Builder) AddSubRoute(path, url string) *Builder {
	child := NewProxyRouteBuilder()
	child.parent = b
	b.children[b.index] = child

	return child.AddRoute(path, url)
}

func (b *Builder) AddSubRouteHandler(path string, forwardHandler http.Handler) *Builder {
	child := NewProxyRouteBuilder()
	child.parent = b
	b.children[b.index] = child

	return child.AddRouteHandler(path, forwardHandler)
}

func (b *Builder) Parent() *Builder {
	if b.parent == nil {
		return b
	}
	return b.parent
}

func (b *Builder) Finish() *Builder {
	return b.Parent()
}

func (b *Builder) currentRoute() *ProxyRoute {
	if b.index < 0 {
		panic("orange-cloudfoundry/gobis/builder: You must add a route by using AddRoute or AddRouteHandler.")
	}
	return b.routes[b.index]
}

func (b *Builder) WithSensitiveHeaders(headers ...string) *Builder {
	rte := b.currentRoute()
	rte.SensitiveHeaders = append(rte.SensitiveHeaders, headers...)
	return b
}

func (b *Builder) WithMethods(methods ...string) *Builder {
	rte := b.currentRoute()
	rte.Methods = append(rte.Methods, methods...)
	return b
}

func (b *Builder) WithHttpProxy(httpProxy, httpsProxy string) *Builder {
	rte := b.currentRoute()
	rte.HttpProxy = httpProxy
	rte.HttpsProxy = httpsProxy
	return b
}

func (b *Builder) WithoutProxy() *Builder {
	rte := b.currentRoute()
	rte.NoProxy = true
	return b
}

func (b *Builder) WithoutBuffer() *Builder {
	rte := b.currentRoute()
	rte.NoBuffer = true
	return b
}

func (b *Builder) WithoutProxyHeaders() *Builder {
	rte := b.currentRoute()
	rte.RemoveProxyHeaders = true
	return b
}

func (b *Builder) WithInsecureSkipVerify() *Builder {
	rte := b.currentRoute()
	rte.InsecureSkipVerify = true
	return b
}

func (b *Builder) WithShowError() *Builder {
	rte := b.currentRoute()
	rte.ShowError = true
	return b
}

func (b *Builder) WithName(name string) *Builder {
	rte := b.currentRoute()
	rte.Name = name
	return b
}

func (b *Builder) WithForwardedHeader(header string) *Builder {
	rte := b.currentRoute()
	rte.ForwardedHeader = header
	return b
}

func (b *Builder) WithMiddlewareParams(params ...interface{}) *Builder {
	rte := b.currentRoute()

	midParams := rte.MiddlewareParams.(map[string]interface{})

	var mapParams map[string]interface{}
	for _, p := range params {
		if reflect.TypeOf(p) != reflect.TypeOf(map[string]interface{}{}) {
			mapParams = InterfaceToMap(p)
		} else {
			mapParams = p.(map[string]interface{})
		}
		for k, v := range mapParams {
			midParams[k] = v
		}
	}
	rte.MiddlewareParams = midParams
	return b
}

func (b *Builder) Build() []ProxyRoute {
	if b.parent != nil {
		return b.parent.Build()
	}
	finalRtes := make([]ProxyRoute, len(b.routes))
	for i, route := range b.routes {
		finalRte := *route
		if _, ok := b.children[i]; ok {
			child := b.children[i]
			child.parent = nil
			finalRte.Routes = child.Build()
		}
		finalRtes[i] = finalRte
	}

	return finalRtes
}
