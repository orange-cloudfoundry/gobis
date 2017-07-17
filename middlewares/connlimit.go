package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	log "github.com/sirupsen/logrus"
	"github.com/vulcand/oxy/connlimit"
	"github.com/vulcand/oxy/utils"
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
	SourceIdentifier string `mapstructure:"source_identifier" json:"source_identifier" yaml:"source_identifier"`
}

func ConnLimit(proxyRoute models.ProxyRoute, handler http.Handler) http.Handler {
	entry := log.WithField("route_name", proxyRoute.Name)
	var config ConnLimitConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding conn limit middleware failed: " + err.Error())
		return handler
	}
	options := config.ConnLimit
	if options == nil || !options.Enable {
		return handler
	}
	if options.SourceIdentifier == "" {
		options.SourceIdentifier = "client.ip"
	}
	if options.Limit == 0 {
		options.Limit = int64(20)
	}
	extractor, err := utils.NewExtractor(options.SourceIdentifier)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding conn limit middleware failed: " + err.Error())
		return handler
	}
	finalHandler, err := connlimit.New(handler, extractor, options.Limit)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding conn limit middleware failed: " + err.Error())
		return handler
	}

	entry.Debug("orange-cloudfoundry/gobis/middlewares:: Adding conn limit middleware.")
	return finalHandler
}
