package gobis_test

import (
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
	"net/http"
	"net/url"
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
		Context("when route doesn't have option ForwardedHeader", func() {
			It("should set request url to upstream url", func() {
				ForwardRequest(ProxyRoute{
					Url: "http://my.proxified.api",
				}, request, "/path")
				Expect(request.URL.String()).Should(Equal("http://my.proxified.api/path"))
			})
			It("should merge query parameters", func() {
				request.URL, _ = url.Parse("http://localhost?key1=val1")
				ForwardRequest(ProxyRoute{
					Url: "http://my.proxified.api?key2=val2",
				}, request, "")
				Expect(request.URL.String()).Should(Equal("http://my.proxified.api?key1=val1&key2=val2"))
			})
			It("should add path to upstream url path", func() {
				ForwardRequest(ProxyRoute{
					Url: "http://my.proxified.api/root",
				}, request, "/path")
				Expect(request.URL.String()).Should(Equal("http://my.proxified.api/root/path"))
			})
			It("should add basic auth when set on url to forward", func() {
				Expect(request.Header.Get("Authorization")).Should(BeEmpty())
				ForwardRequest(ProxyRoute{
					Url: "http://user:password@my.proxified.api",
				}, request, "")
				Expect(request.Header.Get("Authorization")).ShouldNot(BeEmpty())
			})
		})
		Context("when route have option ForwardedHeader", func() {
			It("should set request url to upstream url", func() {
				req, _ := http.NewRequest("GET", "http://localhost/path", nil)
				route := ProxyRoute{
					ForwardedHeader: "X-Forwarded-For",
				}
				req.Header.Set("X-Forwarded-For", "http://my.proxified.api/path")
				ForwardRequest(route, req, route.RequestPath(req))
				Expect(req.URL.String()).Should(Equal("http://my.proxified.api/path"))
			})
			It("should merge query parameters", func() {
				req, _ := http.NewRequest("GET", "http://localhost/path", nil)
				route := ProxyRoute{
					ForwardedHeader: "X-Forwarded-For",
				}
				req.Header.Set("X-Forwarded-For", "http://my.proxified.api?key1=val1&key2=val2")
				ForwardRequest(route, req, route.RequestPath(req))
				Expect(req.URL.String()).Should(Equal("http://my.proxified.api?key1=val1&key2=val2"))
			})
		})
		Context("When username and groups has been set in request context", func() {
			It("should give X-Gobis-Username and X-Gobis-Groups as headers", func() {
				SetUsername(request, "myuser")
				AddGroups(request, "group1", "group2")
				ForwardRequest(ProxyRoute{
					Url: "http://my.proxified.api",
				}, request, "/path")
				Expect(request.Header.Get(XGobisUsername)).Should(Equal("myuser"))
				Expect(request.Header.Get(XGobisGroups)).Should(ContainSubstring("group1"))
				Expect(request.Header.Get(XGobisGroups)).Should(ContainSubstring("group2"))
			})
		})
	})
	Context("CreateMuxRouter", func() {
		Context("when route have option ForwardedHeader set", func() {
			It("should copy get parameter in the request from upstream", func() {
				routes := []ProxyRoute{
					{
						Name:            "app1",
						Path:            "/app1/**",
						ForwardedHeader: "X-Forwarded-Url",
					},
				}
				rtr, err := factory.CreateMuxRouter(routes, "")
				Expect(err).NotTo(HaveOccurred())
				rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
					req, _ := http.NewRequest("GET", "http://localhost/"+routes[0].Name+"/test/toto", nil)
					req.Header.Set("X-Forwarded-Url", "http://localhost/"+routes[0].Name+"/test/toto?data=mydata")
					Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())

					Expect(route.GetName()).Should(Equal(routes[0].Name))
					Expect(req.URL.Query().Get("data")).Should(Equal("mydata"))
					return nil
				})

			})
			Context("without url set in route", func() {
				It("should create a mux router with all routes", func() {
					routes := []ProxyRoute{
						{
							Name:            "app1",
							Path:            "/app1/**",
							ForwardedHeader: "X-Forwarded-Url",
						},
						{
							Name:            "app2",
							Path:            "/app2/*",
							ForwardedHeader: "X-Forwarded-Url",
						},
					}
					rtr, err := factory.CreateMuxRouter(routes, "")
					Expect(err).NotTo(HaveOccurred())
					index := 0
					rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
						req, _ := http.NewRequest("GET", "http://localhost/"+routes[index].Name+"/test/toto", nil)
						req.Header.Set("X-Forwarded-Url", "http://localhost/"+routes[index].Name+"/test/toto")
						if index == 0 {
							Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())
						} else {
							Expect(route.Match(req, &mux.RouteMatch{})).Should(BeFalse())
						}
						Expect(route.GetName()).Should(Equal(routes[index].Name))
						index++
						return nil
					})
					Expect(index).Should(Equal(len(routes)))

				})
			})
			Context("with url set in route", func() {
				It("should create a mux router with all routes by only matching url host when no path is set in url", func() {
					routes := []ProxyRoute{
						{
							Name:            "app1",
							Path:            "/**",
							Url:             "http://localhost",
							ForwardedHeader: "X-Forwarded-Url",
						},
						{
							Name:            "app2",
							Path:            "/**",
							Url:             "http://local.com",
							ForwardedHeader: "X-Forwarded-Url",
						},
					}
					rtr, err := factory.CreateMuxRouter(routes, "")
					Expect(err).NotTo(HaveOccurred())
					index := 0
					rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
						req, _ := http.NewRequest("GET", "http://localhost/"+routes[index].Name+"/test/toto", nil)
						req.Header.Set("X-Forwarded-Url", "http://localhost/"+routes[index].Name+"/test/toto")
						if index == 0 {
							Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())
						} else {
							Expect(route.Match(req, &mux.RouteMatch{})).Should(BeFalse())
						}
						Expect(route.GetName()).Should(Equal(routes[index].Name))
						index++
						return nil
					})
					Expect(index).Should(Equal(len(routes)))

				})
				It("should create a mux router with all routes by matching url host and path when path is set in url", func() {
					routes := []ProxyRoute{
						{
							Name:            "app1",
							Path:            "/**",
							Url:             "http://localhost/app1/**",
							ForwardedHeader: "X-Forwarded-Url",
						},
						{
							Name:            "app2",
							Path:            "/**",
							Url:             "http://localhost/fakepath",
							ForwardedHeader: "X-Forwarded-Url",
						},
					}
					rtr, err := factory.CreateMuxRouter(routes, "")
					Expect(err).NotTo(HaveOccurred())
					index := 0
					rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
						req, _ := http.NewRequest("GET", "http://localhost/"+routes[index].Name+"/test/toto", nil)
						req.Header.Set("X-Forwarded-Url", "http://localhost/"+routes[index].Name+"/test/toto")
						if index == 0 {
							Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())
						} else {
							Expect(route.Match(req, &mux.RouteMatch{})).Should(BeFalse())
						}
						Expect(route.GetName()).Should(Equal(routes[index].Name))
						index++
						return nil
					})
					Expect(index).Should(Equal(len(routes)))

				})
			})
		})
		It("should create a mux router with all routes", func() {
			routes := []ProxyRoute{
				{
					Name: "app1",
					Path: "/app1/**",
					Url:  "http://my.proxified.api",
				},
				{
					Name: "app2",
					Path: "/app2/*",
					Url:  "http://my.second.proxified.api",
				},
			}
			rtr, err := factory.CreateMuxRouter(routes, "")
			Expect(err).NotTo(HaveOccurred())
			index := 0
			rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				u, _ := url.Parse("http://localhost/" + routes[index].Name + "/test/toto")
				req := &http.Request{URL: u}
				if index == 0 {
					Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())
				} else {
					Expect(route.Match(req, &mux.RouteMatch{})).Should(BeFalse())
				}
				Expect(route.GetName()).Should(Equal(routes[index].Name))
				index++
				return nil
			})
			Expect(index).Should(Equal(len(routes)))

		})
		It("should create a mux router methods set if route resquested it", func() {
			routes := []ProxyRoute{
				{
					Name:    "app1",
					Path:    "/app1/**",
					Url:     "http://my.proxified.api",
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
		It("should create a mux router with parent routes", func() {
			parentMuxRouter := mux.NewRouter()
			parentMuxRouter.HandleFunc("/parent", func(w http.ResponseWriter, req *http.Request) {

			})
			routes := []ProxyRoute{
				{
					Name:    "app1",
					Path:    "/app1/**",
					Url:     "http://my.proxified.api",
					Methods: []string{"GET"},
				},
			}
			muxFactory := NewRouterFactoryWithMuxRouter(parentMuxRouter)
			rtr, err := muxFactory.CreateMuxRouter(routes, "")
			Expect(err).NotTo(HaveOccurred())
			var r *mux.Route
			index := 0
			rtr.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				r = route
				tpl, _ := route.GetPathTemplate()
				if index == 0 {
					Expect(tpl).Should(Equal("/parent"))
				} else {
					u, _ := url.Parse("http://localhost/" + routes[index-1].Name + "/test")
					req := &http.Request{URL: u, Method: "GET"}
					Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())
				}
				index++
				return nil
			})
			Expect(index).Should(Equal(2))

		})
	})
	Context("CreateMuxRouterRouteService", func() {
		It("should create a mux router with all routes and the route for forwarded url", func() {
			routes := []ProxyRoute{
				{
					Name: "app1",
					Path: "/app1/**",
					Url:  "http://my.proxified.api",
				},
				{
					Name: "app2",
					Path: "/app2/**",
					Url:  "http://my.second.proxified.api",
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
					u, _ := url.Parse("http://localhost/" + routes[index].Name + "/test")
					req := &http.Request{URL: u}
					Expect(route.Match(req, &mux.RouteMatch{})).Should(BeTrue())
				}
				index++
				return nil
			})
			Expect(index).Should(Equal(len(routes) + 1))

		})
	})
})
