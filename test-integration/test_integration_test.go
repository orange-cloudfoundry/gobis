package test_integration_test

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orange-cloudfoundry/gobis"
	. "github.com/orange-cloudfoundry/gobis/gobistest"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
)

var gobisTestHandler *GobisHandlerTest
var rr *httptest.ResponseRecorder
var _ = BeforeSuite(func() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
})

var _ = BeforeEach(func() {
	rr = httptest.NewRecorder()
})
var _ = AfterEach(func() {
	gobisTestHandler.Close()
})
var _ = AfterSuite(func() {

})

var _ = Describe("TestIntegration", func() {
	Context("simple forwarding", func() {
		var defaultRoute gobis.ProxyRoute
		BeforeEach(func() {
			defaultRoute = gobis.ProxyRoute{
				Name:      "myroute",
				Path:      gobis.NewPathMatcher("/**"),
				Methods:   []string{"GET"},
				ShowError: true,
			}
		})

		It("should not redirect to backend when http method is wrong.", func() {
			gobisTestHandler = NewSimpleGobisHandlerTest(defaultRoute)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(defaultRoute, "POST")
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).ShouldNot(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(404))
		})
		It("should redirect to backend with gobis header", func() {
			defaultRoute.Path = gobis.NewPathMatcher("/anypath")
			gobisTestHandler = NewSimpleGobisHandlerTest(defaultRoute)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer GinkgoRecover()
				Expect(req.Header.Get(gobis.GobisHeaderName)).To(Equal("true"))
				Expect(req.Header).To(HaveKey(gobis.XGobisUsername))
				Expect(req.Header).To(HaveKey(gobis.XGobisGroups))
				Expect(req.Header).To(HaveKey("X-Forwarded-Host"))
				Expect(req.Header).To(HaveKey("X-Forwarded-Proto"))
				Expect(req.Header).To(HaveKey("X-Forwarded-Server"))
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(defaultRoute)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should redirect to backend with gobis header when path has subpath", func() {
			defaultRoute.Path = gobis.NewPathMatcher("/apath/**")
			gobisTestHandler = NewSimpleGobisHandlerTest(defaultRoute)
			routeServer := gobisTestHandler.ServerFirst()
			routeServer.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer GinkgoRecover()
				Expect(req.Header.Get(gobis.GobisHeaderName)).To(Equal("true"))
				Expect(req.Header).To(HaveKey(gobis.XGobisUsername))
				Expect(req.Header).To(HaveKey(gobis.XGobisGroups))
				Expect(req.Header).To(HaveKey("X-Forwarded-Host"))
				Expect(req.Header).To(HaveKey("X-Forwarded-Proto"))
				Expect(req.Header).To(HaveKey("X-Forwarded-Server"))
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(defaultRoute)
			req.URL.Path = "/apath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should redirect to backend without X-Forwarded-* header when user deactivate it", func() {
			myroute := gobis.ProxyRoute{
				Name:               "myroute",
				Path:               gobis.NewPathMatcher("/**"),
				RemoveProxyHeaders: true,
			}
			gobisTestHandler = NewSimpleGobisHandlerTest(myroute)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer GinkgoRecover()
				Expect(req.Header.Get(gobis.GobisHeaderName)).To(Equal("true"))
				Expect(req.Header).To(HaveKey(gobis.XGobisUsername))
				Expect(req.Header).To(HaveKey(gobis.XGobisGroups))
				Expect(req.Header).ToNot(HaveKey("X-Forwarded-Host"))
				Expect(req.Header).ToNot(HaveKey("X-Forwarded-Proto"))
				Expect(req.Header).ToNot(HaveKey("X-Forwarded-Server"))
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(defaultRoute)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should not redirect to backend when path is incorrect in request", func() {
			myroute := gobis.ProxyRoute{
				Name:               "myroute",
				Path:               gobis.NewPathMatcher("/apath/**"),
				RemoveProxyHeaders: true,
			}
			gobisTestHandler = NewSimpleGobisHandlerTest(myroute)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(defaultRoute)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).ShouldNot(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(404))
		})
		It("should show error as json when user set ShowError to true", func() {
			errorHandler := SimpleTestHandleFunc(func(w http.ResponseWriter, req *http.Request, p FakeMiddlewareParams) {
				panic("this is an error")
			})
			gobisTestHandler = NewGobisHandlerTest(
				[]gobis.ProxyRoute{defaultRoute},
				NewFakeMiddleware(errorHandler),
			)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(defaultRoute)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			var jsonError gobis.JsonError
			err = json.Unmarshal(content, &jsonError)
			Expect(err).NotTo(HaveOccurred())

			Expect(jsonError.Details).Should(Equal("this is an error"))
			Expect(jsonError.RouteName).Should(Equal(defaultRoute.Name))
			Expect(resp.StatusCode).Should(Equal(500))
		})
		Context("with multiple routes", func() {
			It("should redirect correctly to url", func() {
				firstRoute := gobis.ProxyRoute{
					Name: "firstRoute",
					Path: gobis.NewPathMatcher("/firstroute/**"),
				}
				secondRoute := gobis.ProxyRoute{
					Name: "secondRoute",
					Path: gobis.NewPathMatcher("/secondroute/**"),
				}
				gobisTestHandler = NewSimpleGobisHandlerTest(firstRoute, secondRoute)
				gobisTestHandler.SetBackendHandler(firstRoute, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("first route"))
				}))
				gobisTestHandler.SetBackendHandler(secondRoute, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("second route"))
				}))

				// first route
				req := CreateRequest(defaultRoute)
				req.URL.Path = "/firstroute"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("first route"))
				Expect(resp.StatusCode).Should(Equal(200))

				// second route
				rr = httptest.NewRecorder()
				req = CreateRequest(defaultRoute)
				req.URL.Path = "/secondroute"
				gobisTestHandler.ServeHTTP(rr, req)
				resp = rr.Result()

				content, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("second route"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
			It("should fallback redirect when first match not correspond and the second is wildcard", func() {
				firstRoute := gobis.ProxyRoute{
					Name: "firstRoute",
					Path: gobis.NewPathMatcher("/firstroute/**"),
				}
				secondRoute := gobis.ProxyRoute{
					Name: "secondRoute",
					Path: gobis.NewPathMatcher("/**"),
				}
				gobisTestHandler = NewSimpleGobisHandlerTest(firstRoute, secondRoute)
				gobisTestHandler.SetBackendHandler(firstRoute, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("route"))
				}))
				gobisTestHandler.SetBackendHandler(secondRoute, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("fallback"))
				}))

				// first route
				req := CreateRequest(defaultRoute)
				req.URL.Path = "/firstroute"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("route"))
				Expect(resp.StatusCode).Should(Equal(200))

				// second route
				rr = httptest.NewRecorder()
				req = CreateRequest(defaultRoute)
				req.URL.Path = "/anypath"
				gobisTestHandler.ServeHTTP(rr, req)
				resp = rr.Result()

				content, err = ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("fallback"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
		})
	})
	Context("chaining forwarding", func() {
		It("should chain to sub request when routes is set inside a route", func() {
			subRoute := gobis.ProxyRoute{
				Name: "subRoute",
				Path: gobis.NewPathMatcher("/sub"),
			}
			route := gobis.ProxyRoute{
				Name:   "parentRoute",
				Path:   gobis.NewPathMatcher("/parent/**"),
				Routes: []gobis.ProxyRoute{subRoute},
			}
			gobisTestHandler = NewSimpleGobisHandlerTest(route)
			gobisTestHandler.SetBackendHandler(route, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("parent"))
			}))
			gobisTestHandler.SetBackendHandler(subRoute, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("sub"))
			}))

			// first route
			req := CreateRequest(route)
			req.URL.Path = "/parent/any"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("parent"))
			Expect(resp.StatusCode).Should(Equal(200))

			// second route
			rr = httptest.NewRecorder()
			req = CreateRequest(subRoute)
			req.URL.Path = "/parent/sub"
			gobisTestHandler.ServeHTTP(rr, req)
			resp = rr.Result()

			content, err = ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("sub"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should apply middleware from middleware from sub and after from parent ", func() {
			midParent := SimpleTestHandleFunc(func(w http.ResponseWriter, req *http.Request, p FakeMiddlewareParams) {
				params := p.TestParams.(map[string]interface{})
				if _, ok := params["parentHeaderKey"]; !ok {
					return
				}
				req.Header.Set(params["parentHeaderKey"].(string), params["parentHeaderValue"].(string))
			})
			midSub := SimpleTestHandleFunc(func(w http.ResponseWriter, req *http.Request, p FakeMiddlewareParams) {
				params := p.TestParams.(map[string]interface{})
				if _, ok := params["subHeaderKey"]; !ok {
					return
				}
				req.Header.Set(params["subHeaderKey"].(string), params["subHeaderValue"].(string))
			})
			subRoute := gobis.ProxyRoute{
				Name: "subRoute",
				Path: gobis.NewPathMatcher("/sub"),
				MiddlewareParams: CreateInlineTestParams(
					"subHeaderKey", "X-Sub-Header",
					"subHeaderValue", "sub",
				),
			}
			route := gobis.ProxyRoute{
				Name:   "parentRoute",
				Path:   gobis.NewPathMatcher("/parent/**"),
				Routes: []gobis.ProxyRoute{subRoute},
				MiddlewareParams: CreateInlineTestParams(
					"parentHeaderKey", "X-Parent-Header",
					"parentHeaderValue", "parent",
				),
			}
			gobisTestHandler = NewGobisHandlerTest(
				[]gobis.ProxyRoute{route},
				NewFakeMiddleware(midParent),
				NewFakeMiddleware(midSub),
			)
			gobisTestHandler.SetBackendHandler(route, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("parent"))
			}))
			gobisTestHandler.SetBackendHandler(subRoute, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer GinkgoRecover()
				Expect(req.Header.Get("X-Parent-Header")).To(Equal(""))
				Expect(req.Header.Get("X-Sub-Header")).To(Equal("sub"))
				w.Write([]byte("sub"))
			}))

			req := CreateRequest(subRoute)
			req.URL.Path = "/parent/sub"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("sub"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
	})
	Context("forwarding with forwarded header", func() {
		var forwardedHeader string = "X-Forward-Url"
		It("should redirect to backend with gobis header", func() {
			route := gobis.ProxyRoute{
				Name:            "myroute",
				Path:            gobis.NewPathMatcher("/**"),
				ForwardedHeader: forwardedHeader,
			}
			gobisTestHandler = NewSimpleGobisHandlerTest(route)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer GinkgoRecover()
				Expect(req.Header.Get(gobis.GobisHeaderName)).To(Equal("true"))
				Expect(req.Header).To(HaveKey(gobis.XGobisUsername))
				Expect(req.Header).To(HaveKey(gobis.XGobisGroups))
				Expect(req.Header).To(HaveKey("X-Forwarded-Host"))
				Expect(req.Header).To(HaveKey("X-Forwarded-Proto"))
				Expect(req.Header).To(HaveKey("X-Forwarded-Server"))
				Expect(req.URL.Path).To(Equal("/mypath"))
				w.Write([]byte("route1 content"))
			}))

			server := gobisTestHandler.ServerFirst()
			req := CreateRequest(route)
			req.Header.Set(forwardedHeader, server.Server.URL+"/mypath")
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should not redirect to backend when not matching path route param", func() {
			route := gobis.ProxyRoute{
				Name:            "myroute",
				Path:            gobis.NewPathMatcher("/forcepath"),
				ForwardedHeader: forwardedHeader,
			}
			gobisTestHandler = NewSimpleGobisHandlerTest(route)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("route1 content"))
			}))

			server := gobisTestHandler.ServerFirst()
			req := CreateRequest(route)
			req.Header.Set(forwardedHeader, server.Server.URL+"/mypath")
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).ShouldNot(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(404))
		})
	})
	Context("when use http(s) proxy", func() {
		It("should use http proxy when is set", func() {
			httpProxy := CreateBackendServer("httpProxy")
			route := gobis.ProxyRoute{
				Name:      "myroute",
				Path:      gobis.NewPathMatcher("/**"),
				HttpProxy: httpProxy.Server.URL,
			}
			httpProxy.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusTemporaryRedirect)
				w.Write([]byte("proxified"))
			}))
			gobisTestHandler = NewSimpleGobisHandlerTest(route)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(route)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("proxified"))
			Expect(resp.StatusCode).Should(Equal(http.StatusTemporaryRedirect))
		})
		It("should use https proxy when is set", func() {
			httpsProxy := CreateBackendServer("httpsProxy")
			route := gobis.ProxyRoute{
				Name:       "myroute",
				Path:       gobis.NewPathMatcher("/**"),
				HttpsProxy: httpsProxy.Server.URL,
			}
			passThroughProxy := false
			httpsProxy.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				passThroughProxy = true
			}))
			gobisTestHandler = NewSimpleGobisHandlerTestInSsl(route)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("route1 content"))
			}))

			req := CreateRequest(route)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)

			Expect(passThroughProxy).Should(BeTrue())
		})
	})
	Context("when use a forward handler in route", func() {
		It("should not use reverse proxy but handler instead", func() {
			route := gobis.ProxyRoute{
				Name: "myroute",
				Path: gobis.NewPathMatcher("/**"),
				ForwardHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("content forward"))
				}),
			}
			gobisTestHandler = NewSimpleGobisHandlerTestInSsl(route)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("should not be seen"))
			}))

			req := CreateRequest(route)
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("content forward"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
	})
	Context("forward with middleware", func() {
		Context("middleware override path", func() {
			It("should use the overrided path when reverse", func() {
				middleware := TestHandlerFunc(func(p HandlerParams) {
					defer GinkgoRecover()
					params := p.Params.TestParams.(map[string]interface{})
					Expect(params["key"]).Should(Equal("value"))
					p.W.Write([]byte("intercepted "))
					gobis.SetPath(p.Req, "/newpath")
					p.Next.ServeHTTP(p.W, p.Req)
				})
				route := gobis.ProxyRoute{
					Name:             "myroute",
					Path:             gobis.NewPathMatcher("/**"),
					MiddlewareParams: CreateInlineTestParams("key", "value"),
					ForwardHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path == "/newpath" {
							w.Write([]byte("forward new path"))
						}
					}),
				}
				gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
				gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))

				req := CreateRequest(route)
				req.URL.Path = "/anypath"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("intercepted forward new path"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
		})
		Context("with options passthrough is true", func() {
			It("should pass through middleware before forward when there is no cors options set", func() {
				middleware := TestHandlerFunc(func(p HandlerParams) {
					defer GinkgoRecover()
					params := p.Params.TestParams.(map[string]interface{})
					Expect(params["key"]).Should(Equal("value"))
					p.W.Write([]byte("intercepted"))
				})
				route := gobis.ProxyRoute{
					Name:               "myroute",
					Path:               gobis.NewPathMatcher("/**"),
					MiddlewareParams:   CreateInlineTestParams("key", "value"),
					OptionsPassthrough: true,
				}
				gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
				gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("forward"))
				}))

				req := CreateRequest(route)
				req.Method = "GET"
				req.URL.Path = "/anypath"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("intercepted"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
			It("should not pass through middleware before forward when there is cors options set", func() {
				middleware := TestHandlerFunc(func(p HandlerParams) {
					defer GinkgoRecover()
					params := p.Params.TestParams.(map[string]interface{})
					Expect(params["key"]).Should(Equal("value"))
					p.W.Write([]byte("intercepted"))
				})
				route := gobis.ProxyRoute{
					Name:               "myroute",
					Path:               gobis.NewPathMatcher("/**"),
					MiddlewareParams:   CreateInlineTestParams("key", "value"),
					OptionsPassthrough: true,
				}
				gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
				gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("forward"))
				}))

				req := CreateRequest(route)
				req.Method = "OPTIONS"
				req.Header.Add("Access-Control-Request-Method", "POST")
				req.URL.Path = "/anypath"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("forward"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
		})
		Context("with host passthrough filled", func() {
			It("should pass through middleware before forward when current host not matching", func() {
				middleware := TestHandlerFunc(func(p HandlerParams) {
					defer GinkgoRecover()
					params := p.Params.TestParams.(map[string]interface{})
					Expect(params["key"]).Should(Equal("value"))
					p.W.Write([]byte("intercepted"))
				})
				route := gobis.ProxyRoute{
					Name:             "myroute",
					Path:             gobis.NewPathMatcher("/**"),
					MiddlewareParams: CreateInlineTestParams("key", "value"),
					HostsPassthrough: []*gobis.HostMatcher{gobis.NewHostMatcher("*.passthrough.com")},
				}
				gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
				gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("forward"))
				}))

				req := CreateRequest(route)
				req.Method = "GET"
				req.URL.Path = "/anypath"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("intercepted"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
			It("should not pass through middleware before forward when there is cors options set", func() {
				middleware := TestHandlerFunc(func(p HandlerParams) {
					defer GinkgoRecover()
					params := p.Params.TestParams.(map[string]interface{})
					Expect(params["key"]).Should(Equal("value"))
					p.W.Write([]byte("intercepted"))
				})
				route := gobis.ProxyRoute{
					Name:             "myroute",
					Path:             gobis.NewPathMatcher("/**"),
					MiddlewareParams: CreateInlineTestParams("key", "value"),
					HostsPassthrough: []*gobis.HostMatcher{gobis.NewHostMatcher("*.app.com")},
				}
				gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
				gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Write([]byte("forward"))
				}))

				req := CreateRequest(route)
				req.URL.Path = "/anypath"
				gobisTestHandler.ServeHTTP(rr, req)
				resp := rr.Result()

				content, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).Should(Equal("forward"))
				Expect(resp.StatusCode).Should(Equal(200))
			})
		})
		It("should pass through middleware before forward", func() {
			middleware := TestHandlerFunc(func(p HandlerParams) {
				defer GinkgoRecover()
				params := p.Params.TestParams.(map[string]interface{})
				Expect(params["key"]).Should(Equal("value"))
				p.W.Write([]byte("intercepted"))
			})
			route := gobis.ProxyRoute{
				Name:             "myroute",
				Path:             gobis.NewPathMatcher("/**"),
				MiddlewareParams: CreateInlineTestParams("key", "value"),
			}
			gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("forward"))
			}))

			req := CreateRequest(route)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("intercepted"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should pass through middleware before forward when middleware params is a struct", func() {
			type Astruct struct {
				Key   string `mapstructure:"key"`
				Value string `mapstructure:"value"`
			}
			middleware := TestHandlerFunc(func(p HandlerParams) {
				defer GinkgoRecover()
				params := p.Params.TestParams.(map[string]interface{})
				Expect(params["key"]).Should(Equal("value"))
				p.W.Write([]byte("intercepted"))
			})
			route := gobis.ProxyRoute{
				Name: "myroute",
				Path: gobis.NewPathMatcher("/**"),
				MiddlewareParams: FakeMiddlewareParams{
					TestParams: map[string]interface{}{
						"key": "value",
					},
				},
			}
			gobisTestHandler = NewGobisHandlerTest([]gobis.ProxyRoute{route}, NewFakeMiddleware(middleware))
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("forward"))
			}))

			req := CreateRequest(route)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("intercepted"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
		It("should have request with username and groups when middleware set it", func() {
			middlewareAuth := TestHandlerFunc(func(p HandlerParams) {
				gobis.SetUsername(p.Req, "me")
				gobis.AddGroups(p.Req, "group1", "group2")
				p.W.Write([]byte("intercepted"))
				p.Next.ServeHTTP(p.W, p.Req)
			})
			middlewareAssert := SimpleTestHandleFunc(func(w http.ResponseWriter, req *http.Request, p FakeMiddlewareParams) {
				Expect(gobis.Username(req)).To(Equal("me"))
				Expect(gobis.Groups(req)).To(ContainElement("group1"))
				Expect(gobis.Groups(req)).To(ContainElement("group2"))
			})
			route := gobis.ProxyRoute{
				Name:             "myroute",
				Path:             gobis.NewPathMatcher("/**"),
				MiddlewareParams: CreateInlineTestParams("key", "value"),
			}
			gobisTestHandler = NewGobisHandlerTest(
				[]gobis.ProxyRoute{route},
				NewFakeMiddleware(middlewareAuth),
				NewFakeMiddleware(middlewareAssert),
			)
			gobisTestHandler.SetBackendHandlerFirst(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer GinkgoRecover()
				Expect(req.Header.Get(gobis.XGobisUsername)).Should(Equal("me"))
				Expect(req.Header.Get(gobis.XGobisGroups)).Should(ContainSubstring("group1"))
				Expect(req.Header.Get(gobis.XGobisGroups)).Should(ContainSubstring("group2"))
				w.Write([]byte("forward"))
			}))

			req := CreateRequest(route)
			req.URL.Path = "/anypath"
			gobisTestHandler.ServeHTTP(rr, req)
			resp := rr.Result()

			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("interceptedforward"))
			Expect(resp.StatusCode).Should(Equal(200))

		})
	})
})
