package gobistest

import (
	"fmt"
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

type SimpleTestHandleFunc func(http.ResponseWriter, *http.Request, MiddlewareTestParams)

func (h SimpleTestHandleFunc) ServeHTTP(p HandlerParams) {
	h(p.W, p.Req, p.Params)
	p.Next.ServeHTTP(p.W, p.Req)
}

type MiddlewareTestParams struct {
	TestParams map[string]interface{} `mapstructure:"test_params" json:"test_params" yaml:"test_params"`
}
type HandlerParams struct {
	W      http.ResponseWriter
	Req    *http.Request
	Params MiddlewareTestParams
	Next   http.Handler
	Route  gobis.ProxyRoute
}
type MiddlewareTest struct {
	testHandler TestHandler
}

func NewMiddlewareTest(testHandler TestHandler) *MiddlewareTest {
	return &MiddlewareTest{testHandler}
}
func (m MiddlewareTest) Handler(route gobis.ProxyRoute, params interface{}, next http.Handler) (http.Handler, error) {
	testParams := params.(MiddlewareTestParams)
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
func (m MiddlewareTest) Schema() interface{} {
	return MiddlewareTestParams{}
}
func CreateInlineParams(elems ...interface{}) map[string]interface{} {
	if len(elems)%2 == 1 {
		panic("Parameters are not in pairs")
	}
	finalParams := make(map[string]interface{})
	var data interface{}
	for i, elem := range elems {
		if (i+1)%2 == 1 {
			data = elem
			continue
		}
		finalParams[fmt.Sprint(data)] = elem
	}
	return CreateParams(finalParams)
}
func CreateParams(params map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"test_params": params,
	}
}
