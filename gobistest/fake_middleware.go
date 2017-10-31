package gobistest

import (
	"github.com/orange-cloudfoundry/gobis"
	"net/http"
)

type TestHandler interface {
	ServeHTTP(HandlerParams)
}
type TestHandlerFunc func(HandlerParams)

func (f TestHandlerFunc) ServeHTTP(p HandlerParams) {
	f(p)
}

type SimpleTestHandleFunc func(http.ResponseWriter, *http.Request, FakeMiddlewareParams)

func (h SimpleTestHandleFunc) ServeHTTP(p HandlerParams) {
	h(p.W, p.Req, p.Params)
	p.Next.ServeHTTP(p.W, p.Req)
}

type FakeMiddlewareParams struct {
	TestParams map[string]interface{} `mapstructure:"test_params" json:"test_params" yaml:"test_params"`
}
type HandlerParams struct {
	W      http.ResponseWriter
	Req    *http.Request
	Params FakeMiddlewareParams
	Next   http.Handler
	Route  gobis.ProxyRoute
}
type FakeMiddleware struct {
	testHandler TestHandler
}

func NewFakeMiddleware(testHandler TestHandler) *FakeMiddleware {
	return &FakeMiddleware{testHandler}
}
func (m FakeMiddleware) Handler(route gobis.ProxyRoute, params interface{}, next http.Handler) (http.Handler, error) {
	testParams := params.(FakeMiddlewareParams)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m.testHandler == nil {
			next.ServeHTTP(w, req)
			return
		}
		m.testHandler.ServeHTTP(HandlerParams{
			W:      w,
			Req:    req,
			Params: testParams,
			Next:   next,
			Route:  route,
		})
	}), nil
}
func (m FakeMiddleware) Schema() interface{} {
	return FakeMiddlewareParams{}
}
