package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	log "github.com/sirupsen/logrus"
	"github.com/vulcand/oxy/trace"
	"os"
)

type TraceConfig struct {
	Trace *TraceOptions `mapstructure:"trace" json:"trace" yaml:"trace"`
}
type TraceOptions struct {
	// enable request and response capture
	Enabled         bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// add request headers to capture
	RequestHeaders  []string `mapstructure:"request_headers" json:"request_headers" yaml:"request_headers"`
	// add response headers to capture
	ResponseHeaders []string `mapstructure:"response_headers" json:"response_headers" yaml:"response_headers"`
}

func Trace(proxyRoute models.ProxyRoute, handler http.Handler) http.Handler {
	var config TraceConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		log.Errorf("orange-cloudfoundry/gobis/middlewares: Adding trace failed: " + err.Error())
		return handler
	}
	options := config.Trace
	if options == nil || !options.Enabled {
		return handler
	}
	traceOptions := make([]trace.Option, 0)
	if len(options.RequestHeaders) == 0 {
		traceOptions = append(traceOptions, trace.RequestHeaders(options.RequestHeaders...))
	}
	if len(options.ResponseHeaders) == 0 {
		traceOptions = append(traceOptions, trace.ResponseHeaders(options.ResponseHeaders...))
	}
	traceHandler, err := trace.New(handler, os.Stdout, traceOptions...)
	if err != nil {
		log.Errorf("orange-cloudfoundry/gobis/middlewares: Adding trace failed: " + err.Error())
		return handler
	}
	log.Debug("orange-cloudfoundry/gobis/middlewares:: Adding trace middleware to capture request.")
	return traceHandler
}
