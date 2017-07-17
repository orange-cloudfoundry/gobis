package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	log "github.com/sirupsen/logrus"
	"github.com/vulcand/oxy/ratelimit"
	"github.com/vulcand/oxy/utils"
	"time"
)

type RateLimitConfig struct {
	RateLimit *RateLimitOptions `mapstructure:"rate_limit" json:"rate_limit" yaml:"rate_limit"`
}
type RateLimitOptions struct {
	// enable rate limit
	Enable           bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// Limit number of requests (default to 5000)
	Limit            int64 `mapstructure:"limit" json:"limit" yaml:"limit"`
	// Number of seconds when the limit will be reset (default to 1800)
	ResetTime        int64 `mapstructure:"reset_time" json:"reset_time" yaml:"reset_time"`
	// Identify request source to limit the source
	// possible value are 'client.ip', 'request.host' or 'request.header.X-My-Header-Name'
	// (default: client.ip)
	SourceIdentifier string `mapstructure:"source_identifier" json:"source_identifier" yaml:"source_identifier"`
}

func RateLimit(proxyRoute models.ProxyRoute, handler http.Handler) http.Handler {
	entry := log.WithField("route_name", proxyRoute.Name)
	var config RateLimitConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding rate limit middleware failed: " + err.Error())
		return handler
	}
	options := config.RateLimit
	if options == nil || !options.Enable {
		return handler
	}
	if options.SourceIdentifier == "" {
		options.SourceIdentifier = "client.ip"
	}
	if options.ResetTime == 0 {
		options.ResetTime = int64(1800)
	}
	if options.Limit == 0 {
		options.Limit = int64(5000)
	}
	extractor, err := utils.NewExtractor(options.SourceIdentifier)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding rate limit middleware failed: " + err.Error())
		return handler
	}
	rateSet := ratelimit.NewRateSet()
	rateSet.Add(time.Second * time.Duration(options.ResetTime), 1, options.Limit)
	limitHandler, err := ratelimit.New(handler, extractor, rateSet)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding rate limit middleware failed: " + err.Error())
		return handler
	}

	entry.Debug("orange-cloudfoundry/gobis/middlewares:: Adding rate limit middleware.")
	return limitHandler
}
