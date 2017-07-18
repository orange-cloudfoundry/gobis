package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	"github.com/vulcand/oxy/connlimit"
)

type ConnLimitConfig struct {
	ConnLimit *ConnLimitOptions `mapstructure:"conn_limit" json:"conn_limit" yaml:"conn_limit"`
}
type ConnLimitOptions struct {
	// enable conn limit middleware
	Enable           bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// Limit number of simultaneous connection (default to 20)
	Limit            int64 `mapstructure:"limit" json:"limit" yaml:"limit"`
	// Identify request source to limit the source
	// possible value are 'client.ip', 'request.host' or 'request.header.X-My-Header-Name'
	// (default: client.ip)
	// if empty and a username exists in context the source will be set to this content (this allow to conn limit by username from auth middleware)
	// for context see: https://godoc.org/github.com/orange-cloudfoundry/gobis/proxy/ctx#Username
	SourceIdentifier string `mapstructure:"source_identifier" json:"source_identifier" yaml:"source_identifier"`
}

func ConnLimit(proxyRoute models.ProxyRoute, handler http.Handler) (http.Handler, error) {
	var config ConnLimitConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		return handler, err
	}
	options := config.ConnLimit
	if options == nil || !options.Enable {
		return handler, nil
	}
	if options.Limit == 0 {
		options.Limit = int64(20)
	}
	extractor, err := NewGobisSourceExtractor(options.SourceIdentifier)
	if err != nil {
		return handler, err
	}
	finalHandler, err := connlimit.New(handler, extractor, options.Limit)
	if err != nil {
		return handler, err
	}
	return finalHandler, nil
}
