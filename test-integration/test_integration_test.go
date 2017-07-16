package test_integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
	"github.com/orange-cloudfoundry/gobis/proxy"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	"github.com/orange-cloudfoundry/gobis/handlers"
	"fmt"
	"net/http/httptest"
	log "github.com/sirupsen/logrus"
	"os"
	"io/ioutil"
)

var originUrl string = "http://local.app.com"
var forwardToUrl string = "http://forward.%s.app.com"

func createForwardUrl(name string) string {
	return fmt.Sprintf(forwardToUrl, name)
}
func createAppUrl(proxyRoute models.ProxyRoute) string {
	proxyRoute.LoadParams()
	return fmt.Sprintf("%s%s", originUrl, proxyRoute.AppPath)
}

var routerFactory proxy.RouterFactory
var _ = BeforeSuite(func() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	httpmock.Activate()
})

var _ = BeforeEach(func() {
	httpmock.Reset()
	routerFactory = proxy.NewRouterFactory()
	routerFactory.(*proxy.RouterFactoryService).CreateTransportFunc = func(proxyRoute models.ProxyRoute) http.RoundTripper {
		return httpmock.DefaultTransport
	}
})

var _ = AfterSuite(func() {
	httpmock.DeactivateAndReset()
})

func responderFromHandler(handler handlers.GobisHandler, respRecorder *httptest.ResponseRecorder) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		handler.ServeHTTP(respRecorder, req)
		res := respRecorder.Result()
		res.Request = req
		return res, nil
	}
}
func responderFromRecorder(respRecorder *httptest.ResponseRecorder) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		res := respRecorder.Result()
		res.Request = req
		return res, nil
	}
}

var _ = Describe("TestIntegration", func() {

	var gobisHandler handlers.GobisHandler
	Context("without start path and forwarded url", func() {
		config := handlers.DefaultHandlerConfig{
			Routes: []models.ProxyRoute{
				{
					Name: "route1",
					Path: "/route1/**",
					NoBuffer: true,
					Url: createForwardUrl("route1"),
				},
				{
					Name: "route2",
					Path: "/route2/**",
					Url: createForwardUrl("route2"),
					NoBuffer: true,
					ExtraParams: map[string]interface{}{
						"cors": map[string]interface{}{
							"allowed_origins": []string{"http://*.app.com"},
						},
					},
				},
			},
		}
		BeforeEach(func() {
			gobisHandler, _ = handlers.NewDefaultHandlerWithRouterFactory(config, routerFactory)
			for _, route := range config.Routes {
				httpmock.RegisterResponder(
					"GET",
					createAppUrl(route),
					responderFromHandler(gobisHandler, httptest.NewRecorder()),
				)
			}

		})
		PIt("should do things", func() {
			route := config.Routes[0]
			rr := httptest.NewRecorder()
			rr.WriteHeader(200)
			rr.WriteString("route1 content")
			httpmock.RegisterResponder(
				"GET",
				createForwardUrl(route.Name),
				responderFromRecorder(rr),
			)
			resp, err := http.Get(createAppUrl(route))
			Expect(err).NotTo(HaveOccurred())
			content, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).Should(Equal("route1 content"))
			Expect(resp.StatusCode).Should(Equal(200))
		})
	})
})
