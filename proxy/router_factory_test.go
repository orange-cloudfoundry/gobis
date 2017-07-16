package proxy_test

import (
	. "github.com/orange-cloudfoundry/gobis/proxy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	"net/url"
	"github.com/gorilla/mux"
)

var _ = Describe("RouterFactory", func() {
	var factory RouterFactory
	BeforeEach(func() {
		factory = NewRouterFactory()
	})
	Context("ForwardRequest", func() {
		request := &http.Request{}
		BeforeEach(func() {
			headers := make(map[string][]string)
			request.Header = http.Header(headers)
			request.URL, _ = url.Parse("http://localhost")
		})
		It("should set request url to forwarded url", func() {
			ForwardRequest(models.ProxyRoute{
				Url: "http://my.proxified.api",
			}, request, "/path")
			Expect(request.URL.String()).Should(Equal("http://my.proxified.api/path"))
		})
		It("should merge query parameters", func() {
			request.URL, _ = url.Parse("http://localhost?key1=val1")
			ForwardRequest(models.ProxyRoute{
				Url: "http://my.proxified.api?key2=val2",
			}, request, "")
			Expect(request.URL.String()).Should(Equal("http://my.proxified.api?key1=val1&key2=val2"))
		})
		It("should add path to forwarded url path", func() {
			ForwardRequest(models.ProxyRoute{
				Url: "http://my.proxified.api/root",
			}, request, "/path")
			Expect(request.URL.String()).Should(Equal("http://my.proxified.api/root/path"))
		})
		It("should add basic auth when set on url to forward", func() {
			Expect(request.Header.Get("Authorization")).Should(BeEmpty())
			ForwardRequest(models.ProxyRoute{
				Url: "http://user:password@my.proxified.api",
			}, request, "")
			Expect(request.Header.Get("Authorization")).ShouldNot(BeEmpty())
		})
	})
	Context("CreateMuxRouter", func() {
		It("should create a mux router with all routes", func() {
			routes := []models.ProxyRoute{
				{
					Path: "/app1/**",
					Url: "http://my.proxified.api",
				},
				{
					Path: "/app2/**",
					Url: "http://my.second.proxified.api",
				},
			}
			rtr, err := factory.CreateMuxRouter(routes, "")
			Expect(err).NotTo(HaveOccurred())
			index := 0
			rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				tpl, _ := route.GetPathTemplate()
				Expect(tpl).Should(Equal(routes[index].MuxRoute()))
				index++
				return nil
			})
			Expect(index).Should(Equal(len(routes)))

		})
		It("should create a mux router methods set if route resquested it", func() {
			routes := []models.ProxyRoute{
				{
					Path: "/app1/**",
					Url: "http://my.proxified.api",
					Methods: []string{"GET"},
				},
			}
			rtr, err := factory.CreateMuxRouter(routes, "")
			Expect(err).NotTo(HaveOccurred())
			var r *mux.Route
			rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				r = route
				return nil
			})
			methods, _ := r.GetMethods()
			Expect(methods).Should(HaveLen(1))
			Expect(methods[0]).Should(Equal("GET"))

		})
	})
	Context("CreateMuxRouterRouteService", func() {
		It("should create a mux router with all routes and the route for forwarded url", func() {
			routes := []models.ProxyRoute{
				{
					Path: "/app1/**",
					Url: "http://my.proxified.api",
				},
				{
					Path: "/app2/**",
					Url: "http://my.second.proxified.api",
				},
			}
			fwdUrl, _ := url.Parse("http://myapp.local/path")
			rtr, err := factory.CreateMuxRouterRouteService(routes, "", fwdUrl)
			Expect(err).NotTo(HaveOccurred())
			index := 0
			rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				tpl, _ := route.GetPathTemplate()
				if index == len(routes) {
					Expect(tpl).Should(Equal(fwdUrl.Path))
				} else {
					Expect(tpl).Should(Equal(routes[index].MuxRoute()))
				}
				index++
				return nil
			})
			Expect(index).Should(Equal(len(routes) + 1))

		})
	})
})
