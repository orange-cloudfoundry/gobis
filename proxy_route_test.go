package gobis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
	"encoding/json"
	"gopkg.in/yaml.v2"
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
