package gobis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"net/http"
)

var _ = Describe("ProxyRoute", func() {
	Context("UnmarshallJSON", func() {
		It("should complain when check not passing", func() {
			var route ProxyRoute
			jsonRoute := `{"path": "/app/**", "url": "http://my.proxified.api"}`
			err := json.Unmarshal([]byte(jsonRoute), &route)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("name"))
		})
		It("should write app path", func() {
			var route ProxyRoute
			jsonRoute := `{"name": "myroute", "path": "/app/**", "url": "http://my.proxified.api"}`
			err := json.Unmarshal([]byte(jsonRoute), &route)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(route.AppPath).Should(Equal("/app"))
		})
	})
	Context("UnmarshallYAML", func() {
		It("should complain when check not passing", func() {
			var route ProxyRoute
			yamlRoute := "path: /app/**\nurl: http://my.proxified.api"
			err := yaml.Unmarshal([]byte(yamlRoute), &route)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("name"))
		})
		It("should write app path", func() {
			var route ProxyRoute
			yamlRoute := "name: myroute\npath: /app/**\nurl: http://my.proxified.api"
			err := yaml.Unmarshal([]byte(yamlRoute), &route)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(route.AppPath).Should(Equal("/app"))
		})
	})
	Context("UpstreamUrl", func() {
		It("should return original request url if option ForwardedHeader not set", func() {
			route := ProxyRoute{
				Path: "/app/**",
				Url: "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			upstreamUrl := route.UpstreamUrl(req)
			Expect(upstreamUrl.String()).Should(Equal("http://my.proxified.api"))
		})
		It("should return original request url if option ForwardedHeader is set but not found in request", func() {
			route := ProxyRoute{
				Path: "/app/**",
				ForwardedHeader: "X-Forwarded-Url",
				Url: "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			upstreamUrl := route.UpstreamUrl(req)
			Expect(upstreamUrl.String()).Should(Equal("http://my.proxified.api"))
		})
		It("should return url without path from header set in option ForwardedHeader if it's set and found", func() {
			route := ProxyRoute{
				Path: "/app/**",
				ForwardedHeader: "X-Forwarded-Url",
				Url: "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			req.Header.Set("X-Forwarded-Url", "http://other.url.com/otherpath")
			upstreamUrl := route.UpstreamUrl(req)
			Expect(upstreamUrl.String()).Should(Equal("http://other.url.com"))
		})
	})
	Context("RequestPath", func() {
		It("should return original request path if option ForwardedHeader not set", func() {
			route := ProxyRoute{
				Path: "/app/**",
				Url: "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			path := route.RequestPath(req)
			Expect(path).Should(Equal("/path"))
		})
		It("should return original request path if option ForwardedHeader is set but not found in request", func() {
			route := ProxyRoute{
				Path: "/app/**",
				ForwardedHeader: "X-Forwarded-Url",
				Url: "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			path := route.RequestPath(req)
			Expect(path).Should(Equal("/path"))
		})
		It("should return path from url found in header set in option ForwardedHeader if it's set and found", func() {
			route := ProxyRoute{
				Path: "/app/**",
				ForwardedHeader: "X-Forwarded-Url",
				Url: "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			req.Header.Set("X-Forwarded-Url", "http://other.url.com/otherpath")
			path := route.RequestPath(req)
			Expect(path).Should(Equal("/otherpath"))
		})
	})
	Context("Check", func() {
		It("should complain when no name is provided", func() {
			route := ProxyRoute{
				Path: "/app/**",
				Url: "http://my.proxified.api",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("name"))
		})
		It("should complain when no path is provided", func() {
			route := ProxyRoute{
				Name: "myroute",
				Url: "http://my.proxified.api",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("path"))
		})
		It("should complain when no url is provided", func() {
			route := ProxyRoute{
				Path: "/app/**",
				Name: "my route",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("url"))
		})
		It("should complain if url is set to localhost", func() {
			route := ProxyRoute{
				Name: "my route",
				Path: "/app/**",
				Url: "http://localhost",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("localhost or 127.0.0.1"))
		})
		It("should complain if url is set to 127.0.0.1", func() {
			route := ProxyRoute{
				Name: "my route",
				Path: "/app/**",
				Url: "http://127.0.0.1",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("localhost or 127.0.0.1"))
		})
		Context("Check path", func() {
			It("should match /**", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: "/**",
					Url: "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(route.RouteMatcher()).Should(Equal("(/.*)?$"))
			})
			It("should match /*", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: "/*",
					Url: "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(route.RouteMatcher()).Should(Equal("(/[^/]*)?$"))
			})
			It("should not match /app/*", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: "/app/*",
					Url: "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(route.RouteMatcher()).Should(Equal("/app(/[^/]*)?$"))
			})
			It("should not match /app/**", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: "/app/**",
					Url: "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(route.RouteMatcher()).Should(Equal("/app(/.*)?$"))
			})
			It("should not match /*/app", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: "/*/app",
					Url: "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).Should(HaveOccurred())

			})
			It("should not match /app/***", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: "/app/***",
					Url: "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).Should(HaveOccurred())

			})
		})
	})

})
