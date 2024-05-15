package gobis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orange-cloudfoundry/gobis"
	"net/http"
	"net/url"
	"os"
)

var _ = Describe("RouteTransport", func() {
	var fakeUrl *url.URL
	BeforeEach(func() {
		os.Setenv("https_proxy", "http://https.env.proxy.local")
		os.Unsetenv("http_proxy")
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("HTTPS_PROXY")
		SetProtectedHeaders(make([]string, 0))
		fakeUrl, _ = url.Parse("http://fake.url.local/path")
	})
	Context("ProxyFromRouteOrEnv", func() {
		Context("With http proxies set in route", func() {
			It("should use http proxy when request is in http", func() {
				rt := NewRouteTransport(ProxyRoute{
					HttpProxy:  "http://http.proxy.local",
					HttpsProxy: "http://https.proxy.local",
				}).(*RouteTransport)
				proxyUrl, err := rt.ProxyFromRouteOrEnv(&http.Request{
					URL: fakeUrl,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(proxyUrl.String()).Should(Equal("http://http.proxy.local"))
			})
			It("should use https proxy when request is in https", func() {
				rt := NewRouteTransport(ProxyRoute{
					HttpProxy:  "http://http.proxy.local",
					HttpsProxy: "http://https.proxy.local",
				}).(*RouteTransport)
				fakeUrl.Scheme = "https"
				proxyUrl, err := rt.ProxyFromRouteOrEnv(&http.Request{
					URL: fakeUrl,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(proxyUrl.String()).Should(Equal("http://https.proxy.local"))
			})
		})
		Context("Without http proxies set in route", func() {
			It("shouldn't use proxy when no proxies from env var", func() {
				rt := NewRouteTransport(ProxyRoute{}).(*RouteTransport)
				proxyUrl, err := rt.ProxyFromRouteOrEnv(&http.Request{
					URL: fakeUrl,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(proxyUrl).Should(BeNil())
			})
			It("should use a http proxy when there is proxies from env var", func() {
				fakeUrl.Scheme = "https"
				rt := NewRouteTransport(ProxyRoute{}).(*RouteTransport)
				proxyUrl, err := rt.ProxyFromRouteOrEnv(&http.Request{
					URL: fakeUrl,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(proxyUrl.String()).Should(Equal("http://https.env.proxy.local"))
			})
		})
		Context("With no proxy parameter set in route", func() {
			It("shouldn't use proxy", func() {
				fakeUrl.Scheme = "https"
				rt := NewRouteTransport(ProxyRoute{
					NoProxy: true,
				}).(*RouteTransport)
				proxyUrl, err := rt.ProxyFromRouteOrEnv(&http.Request{
					URL: fakeUrl,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(proxyUrl).Should(BeNil())
			})
		})
	})
	Context("TransformRequest", func() {
		request := &http.Request{}
		BeforeEach(func() {
			request.Host = "localhost"
			headers := make(map[string][]string)
			request.Header = headers
			request.Header.Add(XForwardedHost, "localhost")
		})
		It("Shouldn't had X-Forwarded-Host header to request when route remove it", func() {
			Expect(request.Header.Get("X-Forwarded-Host")).Should(Equal("localhost"))
			rt := NewRouteTransport(ProxyRoute{
				RemoveProxyHeaders: true,
			}).(*RouteTransport)
			rt.TransformRequest(request)
			Expect(request.Header.Get("X-Forwarded-Host")).Should(Equal(""))
		})
		It("Should remove sensitive headers to request when route ask for it", func() {
			request.Header.Add("X-Header-First", "1")
			request.Header.Add("X-Header-Second", "2")
			request.Header.Add("X-Header-Third", "3")
			rt := NewRouteTransport(ProxyRoute{
				RemoveProxyHeaders: true,
				SensitiveHeaders:   []string{"X-Header-First", "X-Header-Second"},
			}).(*RouteTransport)
			rt.TransformRequest(request)
			Expect(request.Header.Get("X-Header-First")).Should(Equal(""))
			Expect(request.Header.Get("X-Header-Second")).Should(Equal(""))
			Expect(request.Header.Get("X-Header-Third")).Should(Equal("3"))
		})
		It("Should not remove sensitive headers which are prevented from deletion to request when route ask for it", func() {
			SetProtectedHeaders([]string{"X-Header-First"})
			request.Header.Add("X-Header-First", "1")
			request.Header.Add("X-Header-Second", "2")
			request.Header.Add("X-Header-Third", "3")
			rt := NewRouteTransport(ProxyRoute{
				RemoveProxyHeaders: true,
				SensitiveHeaders:   []string{"X-Header-First", "X-Header-Second"},
			}).(*RouteTransport)
			rt.TransformRequest(request)
			Expect(request.Header.Get("X-Header-First")).Should(Equal("1"))
			Expect(request.Header.Get("X-Header-Second")).Should(Equal(""))
			Expect(request.Header.Get("X-Header-Third")).Should(Equal("3"))
		})
	})
})
