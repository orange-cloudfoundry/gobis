package gobis_test

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
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
		It("should create route when check pass", func() {
			var route ProxyRoute
			jsonRoute := `{"path": "/app/**", "url": "http://my.proxified.api", "name": "myroute"}`
			err := json.Unmarshal([]byte(jsonRoute), &route)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(route.Path.String()).Should(Equal("/app/**"))
			Expect(route.Url).Should(Equal("http://my.proxified.api"))
			Expect(route.Name).Should(Equal("myroute"))
		})
		Context("with hosts passthrough set", func() {
			It("should create route when check pass", func() {
				var route ProxyRoute
				yamlRoute := `{
"path": "/app/**", 
"url": "http://my.proxified.api", 
"name": "myroute",
"hosts_passthrough": ["myhost.com", "*.myhost.com"]}`
				err := yaml.Unmarshal([]byte(yamlRoute), &route)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(route.HostsPassthrough).Should(HaveLen(2))
				Expect(route.HostsPassthrough[0].String()).Should(Equal("myhost.com"))
				Expect(route.HostsPassthrough[1].String()).Should(Equal("*.myhost.com"))
			})
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
		It("should create route when check pass", func() {
			var route ProxyRoute
			yamlRoute := `path: /app/**
url: http://my.proxified.api
name: myroute`
			err := yaml.Unmarshal([]byte(yamlRoute), &route)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(route.Path.String()).Should(Equal("/app/**"))
			Expect(route.Url).Should(Equal("http://my.proxified.api"))
			Expect(route.Name).Should(Equal("myroute"))
		})
		Context("with hosts passthrough set", func() {
			It("should create route when check pass", func() {
				var route ProxyRoute
				yamlRoute := `path: /app/**
url: http://my.proxified.api
name: myroute
hosts_passthrough:
- "myhost.com"
- "*.myhost.com"`
				err := yaml.Unmarshal([]byte(yamlRoute), &route)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(route.HostsPassthrough).Should(HaveLen(2))
				Expect(route.HostsPassthrough[0].String()).Should(Equal("myhost.com"))
				Expect(route.HostsPassthrough[1].String()).Should(Equal("*.myhost.com"))
			})
		})
	})
	Context("UpstreamUrl", func() {
		It("should return original request url if option ForwardedHeader not set", func() {
			route := ProxyRoute{
				Path: NewPathMatcher("/app/**"),
				Url:  "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			upstreamUrl := route.UpstreamUrl(req)
			Expect(upstreamUrl.String()).Should(Equal("http://my.proxified.api"))
		})
		It("should return original request url if option ForwardedHeader is set but not found in request", func() {
			route := ProxyRoute{
				Path:            NewPathMatcher("/app/**"),
				ForwardedHeader: "X-Forwarded-Url",
				Url:             "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			upstreamUrl := route.UpstreamUrl(req)
			Expect(upstreamUrl.String()).Should(Equal("http://my.proxified.api"))
		})
		It("should return url without path from header set in option ForwardedHeader if it's set and found", func() {
			route := ProxyRoute{
				Path:            NewPathMatcher("/app/**"),
				ForwardedHeader: "X-Forwarded-Url",
				Url:             "http://my.proxified.api",
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
				Path: NewPathMatcher("/app/**"),
				Url:  "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			path := route.RequestPath(req)
			Expect(path).Should(Equal("/path"))
		})
		It("should return original request path if option ForwardedHeader is set but not found in request", func() {
			route := ProxyRoute{
				Path:            NewPathMatcher("/app/**"),
				ForwardedHeader: "X-Forwarded-Url",
				Url:             "http://my.proxified.api",
			}
			req, _ := http.NewRequest("GET", "http://localhost.com/path", nil)
			path := route.RequestPath(req)
			Expect(path).Should(Equal("/path"))
		})
		It("should return path from url found in header set in option ForwardedHeader if it's set and found", func() {
			route := ProxyRoute{
				Path:            NewPathMatcher("/app/**"),
				ForwardedHeader: "X-Forwarded-Url",
				Url:             "http://my.proxified.api",
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
				Path: NewPathMatcher("/app/**"),
				Url:  "http://my.proxified.api",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("name"))
		})
		It("should complain when no path is provided", func() {
			route := ProxyRoute{
				Name: "myroute",
				Url:  "http://my.proxified.api",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("path"))
		})
		It("should complain when no url is provided", func() {
			route := ProxyRoute{
				Path: NewPathMatcher("/app/**"),
				Name: "my route",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("URL"))
		})
		It("should complain if url is set to localhost", func() {
			route := ProxyRoute{
				Name: "my route",
				Path: NewPathMatcher("/app/**"),
				Url:  "http://localhost",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("localhost or 127.0.0.1"))
		})
		It("should complain if url is set to 127.0.0.1", func() {
			route := ProxyRoute{
				Name: "my route",
				Path: NewPathMatcher("/app/**"),
				Url:  "http://127.0.0.1",
			}
			err := route.Check()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("localhost or 127.0.0.1"))
		})
		Context("Check path", func() {
			It("should match /**", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: NewPathMatcher("/**"),
					Url:  "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				matcher := route.RouteMatcher()
				Expect(matcher.MatchString("/firstlevel")).Should(BeTrue())
				Expect(matcher.MatchString("/firstlevel/secondlevel")).Should(BeTrue())
			})
			It("should match /*", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: NewPathMatcher("/*"),
					Url:  "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				matcher := route.RouteMatcher()
				Expect(matcher.MatchString("/app")).Should(BeTrue())
				Expect(matcher.MatchString("/app/secondlevel")).Should(BeFalse())
			})
			It("should not match /app/*", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: NewPathMatcher("/app/*"),
					Url:  "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				matcher := route.RouteMatcher()
				Expect(matcher.MatchString("/foo")).Should(BeFalse())
				Expect(matcher.MatchString("/app")).Should(BeTrue())
				Expect(matcher.MatchString("/app/secondlevel")).Should(BeTrue())
				Expect(matcher.MatchString("/app/secondlevel/thirdlevel")).Should(BeFalse())
			})
			It("should not match /app/**", func() {
				route := ProxyRoute{
					Name: "my route",
					Path: NewPathMatcher("/app/**"),
					Url:  "http://my.proxified.api",
				}
				err := route.Check()
				Expect(err).ShouldNot(HaveOccurred())
				matcher := route.RouteMatcher()
				Expect(matcher.MatchString("/foo")).Should(BeFalse())
				Expect(matcher.MatchString("/app")).Should(BeTrue())
				Expect(matcher.MatchString("/app/secondlevel")).Should(BeTrue())
				Expect(matcher.MatchString("/app/secondlevel/thirdlevel")).Should(BeTrue())
			})
			It("should not match /*/app", func() {
				Expect(func() { NewPathMatcher("/*/app") }).Should(Panic())
			})
			It("should not match /app/***", func() {
				Expect(func() { NewPathMatcher("/app/***") }).Should(Panic())
			})
		})
	})

})
