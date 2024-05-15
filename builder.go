package gobis

import (
	"github.com/google/uuid"
	"net/http"
	"reflect"
)

type ProxyRouteBuilder struct {
	routes   []*ProxyRoute
	children map[int]*ProxyRouteBuilder
	parent   *ProxyRouteBuilder
	index    int
}

func Builder() *ProxyRouteBuilder {
	return &ProxyRouteBuilder{
		routes:   make([]*ProxyRoute, 0),
		children: make(map[int]*ProxyRouteBuilder),
		index:    -1,
	}
}

func (b *ProxyRouteBuilder) AddRoute(path, url string) *ProxyRouteBuilder {
	b.routes = append(b.routes, &ProxyRoute{
		Name:             uuid.NewString(),
		Path:             NewPathMatcher(path),
		Url:              url,
		SensitiveHeaders: []string{},
		Methods:          []string{},
		MiddlewareParams: map[string]interface{}{},
		HostsPassthrough: []*HostMatcher{},
	})
	b.index++
	return b
}

func (b *ProxyRouteBuilder) AddRouteHandler(path string, forwardHandler http.Handler) *ProxyRouteBuilder {
	b.routes = append(b.routes, &ProxyRoute{
		Name:             uuid.NewString(),
		Path:             NewPathMatcher(path),
		ForwardHandler:   forwardHandler,
		SensitiveHeaders: []string{},
		Methods:          []string{},
		MiddlewareParams: map[string]interface{}{},
		HostsPassthrough: []*HostMatcher{},
	})
	b.index++
	return b
}

func (b *ProxyRouteBuilder) AddSubRoute(path, url string) *ProxyRouteBuilder {
	child := Builder()
	child.parent = b
	b.children[b.index] = child

	return child.AddRoute(path, url)
}

func (b *ProxyRouteBuilder) AddSubRouteHandler(path string, forwardHandler http.Handler) *ProxyRouteBuilder {
	child := Builder()
	child.parent = b
	b.children[b.index] = child

	return child.AddRouteHandler(path, forwardHandler)
}

func (b *ProxyRouteBuilder) Parent() *ProxyRouteBuilder {
	if b.parent == nil {
		return b
	}
	return b.parent
}

func (b *ProxyRouteBuilder) Finish() *ProxyRouteBuilder {
	return b.Parent()
}

func (b *ProxyRouteBuilder) currentRoute() *ProxyRoute {
	if b.index < 0 {
		panic("orange-cloudfoundry/gobis/builder: You must add a route by using AddRoute or AddRouteHandler.")
	}
	return b.routes[b.index]
}

func (b *ProxyRouteBuilder) WithSensitiveHeaders(headers ...string) *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.SensitiveHeaders = append(rte.SensitiveHeaders, headers...)
	return b
}

func (b *ProxyRouteBuilder) AddHostPassthrough(hostsOrWildcards ...string) *ProxyRouteBuilder {
	rte := b.currentRoute()
	hostMatchers := make([]*HostMatcher, len(hostsOrWildcards))
	for i, hostOrWildcard := range hostsOrWildcards {
		hostMatchers[i] = NewHostMatcher(hostOrWildcard)
	}
	rte.HostsPassthrough = append(rte.HostsPassthrough, hostMatchers...)
	return b
}

func (b *ProxyRouteBuilder) WithMethods(methods ...string) *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.Methods = append(rte.Methods, methods...)
	return b
}

func (b *ProxyRouteBuilder) WithHttpProxy(httpProxy, httpsProxy string) *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.HttpProxy = httpProxy
	rte.HttpsProxy = httpsProxy
	return b
}

func (b *ProxyRouteBuilder) WithoutProxy() *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.NoProxy = true
	return b
}

func (b *ProxyRouteBuilder) WithoutBuffer() *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.NoBuffer = true
	return b
}

func (b *ProxyRouteBuilder) WithoutProxyHeaders() *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.RemoveProxyHeaders = true
	return b
}

func (b *ProxyRouteBuilder) WithInsecureSkipVerify() *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.InsecureSkipVerify = true
	return b
}

func (b *ProxyRouteBuilder) WithOptionsPassthrough() *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.OptionsPassthrough = true
	return b
}

func (b *ProxyRouteBuilder) WithShowError() *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.ShowError = true
	return b
}

func (b *ProxyRouteBuilder) WithName(name string) *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.Name = name
	return b
}

func (b *ProxyRouteBuilder) WithForwardedHeader(header string) *ProxyRouteBuilder {
	rte := b.currentRoute()
	rte.ForwardedHeader = header
	return b
}

func (b *ProxyRouteBuilder) WithMiddlewareParams(params ...interface{}) *ProxyRouteBuilder {
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

func (b *ProxyRouteBuilder) Build() []ProxyRoute {
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
