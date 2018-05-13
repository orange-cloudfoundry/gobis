package gobis_test

import (
	. "github.com/orange-cloudfoundry/gobis"
	. "github.com/onsi/gomega"
	. "github.com/onsi/ginkgo"
	"net/http"
)

var _ = Describe("Builder", func() {
	var builder *Builder
	BeforeEach(func() {
		builder = NewProxyRouteBuilder()
	})
	Context("Create simle route", func() {
		It("should create all fields on a single route", func() {
			routes := builder.AddRoute("/aroute", "http://url.com").
				WithHttpProxy("http://proxy.com", "https://proxy.com").
				WithInsecureSkipVerify().
				WithMethods("GET").
				WithName("aname").
				WithForwardedHeader("X-Forward").
				WithoutBuffer().
				WithoutProxy().
				WithoutProxyHeaders().
				WithSensitiveHeaders("X-My-Header").
				WithShowError().
				Build()

			finalRte := routes[0]
			Expect(finalRte.Name).Should(Equal("aname"))
			Expect(finalRte.ForwardedHeader).Should(Equal("X-Forward"))
			Expect(finalRte.ForwardHandler).Should(BeNil())
			Expect(finalRte.Path).Should(Equal("/aroute"))
			Expect(finalRte.Url).Should(Equal("http://url.com"))
			Expect(finalRte.InsecureSkipVerify).Should(BeTrue())
			Expect(finalRte.NoBuffer).Should(BeTrue())
			Expect(finalRte.NoProxy).Should(BeTrue())
			Expect(finalRte.RemoveProxyHeaders).Should(BeTrue())
			Expect(finalRte.ShowError).Should(BeTrue())
			Expect(finalRte.Methods[0]).Should(Equal("GET"))
			Expect(finalRte.SensitiveHeaders[0]).Should(Equal("X-My-Header"))
		})
		It("should create with forward handler when given", func() {
			routes := builder.AddRouteHandler("/aroute", http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {

			})).Build()

			finalRte := routes[0]
			Expect(finalRte.Name).ShouldNot(BeEmpty())
			Expect(finalRte.ForwardHandler).ShouldNot(BeNil())
			Expect(finalRte.Path).Should(Equal("/aroute"))
		})
	})

	Context("calling method without create route", func() {
		It("should raise an error", func() {
			Expect(func() { builder.WithShowError() }).Should(Panic())
		})
	})

	Context("adding middleware params", func() {
		It("should be able to deserialize interface to map", func() {
			aStruct := struct {
				Name string
			}{"value"}
			routes := builder.AddRoute("/apath", "http://url.com").
				WithMiddlewareParams(aStruct).
				Build()

			rte := routes[0]
			params := rte.MiddlewareParams.(map[string]interface{})
			Expect(params["Name"]).Should(Equal("value"))
		})
		It("should accept map as it is", func() {
			routes := builder.AddRoute("/apath", "http://url.com").
				WithMiddlewareParams(map[string]interface{}{"name": "value"}).
				Build()

			rte := routes[0]
			params := rte.MiddlewareParams.(map[string]interface{})
			Expect(params["name"]).Should(Equal("value"))
		})
		It("should be able mix map and interface", func() {
			aStruct := struct {
				Foo string
			}{"value"}

			routes := builder.AddRoute("/apath", "http://url.com").
				WithMiddlewareParams(map[string]interface{}{"bar": "value"}).
				WithMiddlewareParams(aStruct).
				Build()

			rte := routes[0]
			params := rte.MiddlewareParams.(map[string]interface{})
			Expect(params["bar"]).Should(Equal("value"))
			Expect(params["Foo"]).Should(Equal("value"))
		})
	})

	Context("create multiple route", func() {
		It("should create all first level routes", func() {
			routes := builder.
				AddRoute("/1", "http://url1.com").WithName("1").
				AddRoute("/2", "http://url2.com").WithName("2").
				Build()

			Expect(routes[0].Name).Should(Equal("1"))
			Expect(routes[0].Url).Should(Equal("http://url1.com"))
			Expect(routes[0].Path).Should(Equal("/1"))

			Expect(routes[1].Name).Should(Equal("2"))
			Expect(routes[1].Url).Should(Equal("http://url2.com"))
			Expect(routes[1].Path).Should(Equal("/2"))
		})
		It("should create all first level routes with all sub routes", func() {
			routes := builder.
				AddRoute("/1", "http://url1.com").WithName("1").
				AddSubRoute("/sub", "").Finish().

				AddRoute("/2", "http://url2.com").WithName("2").
				Build()

			Expect(routes[0].Name).Should(Equal("1"))
			Expect(routes[0].Url).Should(Equal("http://url1.com"))
			Expect(routes[0].Path).Should(Equal("/1"))
			Expect(routes[0].Routes[0].Path).Should(Equal("/sub"))

			Expect(routes[1].Name).Should(Equal("2"))
			Expect(routes[1].Url).Should(Equal("http://url2.com"))
			Expect(routes[1].Path).Should(Equal("/2"))
			Expect(routes[1].Routes).Should(HaveLen(0))
		})
	})
})
