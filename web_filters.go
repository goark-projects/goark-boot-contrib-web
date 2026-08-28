package gbcweb

import (
	"strconv"
	"strings"
	"time"

	goarkcontainer "goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	goweb "goark.dev/goark/web"
	gowebcors "goark.dev/goark/web/cors"
	gowebfilter "goark.dev/goark/web/filter"
)

const (
	propertySpringWebCORSEnabled               = "spring.web.cors.enabled"
	propertySpringWebCORSAllowedOrigins        = "spring.web.cors.allowed-origins"
	propertySpringWebCORSAllowedOriginPatterns = "spring.web.cors.allowed-origin-patterns"
	propertySpringWebCORSAllowedMethods        = "spring.web.cors.allowed-methods"
	propertySpringWebCORSAllowedHeaders        = "spring.web.cors.allowed-headers"
	propertySpringWebCORSExposedHeaders        = "spring.web.cors.exposed-headers"
	propertySpringWebCORSAllowCredentials      = "spring.web.cors.allow-credentials"
	propertySpringWebCORSMaxAge                = "spring.web.cors.max-age"
	propertySpringMVCCORSEnabled               = "spring.mvc.cors.enabled"
	propertySpringMVCCORSAllowedOrigins        = "spring.mvc.cors.allowed-origins"
	propertySpringMVCCORSAllowedOriginPatterns = "spring.mvc.cors.allowed-origin-patterns"
	propertySpringMVCCORSAllowedMethods        = "spring.mvc.cors.allowed-methods"
	propertySpringMVCCORSAllowedHeaders        = "spring.mvc.cors.allowed-headers"
	propertySpringMVCCORSExposedHeaders        = "spring.mvc.cors.exposed-headers"
	propertySpringMVCCORSAllowCredentials      = "spring.mvc.cors.allow-credentials"
	propertySpringMVCCORSMaxAge                = "spring.mvc.cors.max-age"
	propertySpringForwardedHeadersEnabled      = "server.forward-headers-strategy"
	propertySpringShallowETagEnabled           = "spring.web.shallow-etag.enabled"
	propertySpringShallowETagMaxBodyBytes      = "spring.web.shallow-etag.max-body-bytes"
	orderForwardedHeadersFilter                = -300
	orderCORSFilter                            = -200
	orderShallowETagFilter                     = 100
)

type filterSettings struct {
	cors             corsFilterSettings
	forwardedHeaders forwardedHeadersSettings
	shallowETag      shallowETagSettings
}

type corsFilterSettings struct {
	enabled    bool
	enabledSet bool
	config     gowebcors.Config
}

type forwardedHeadersSettings struct {
	enabled bool
}

type shallowETagSettings struct {
	enabled      bool
	maxBodyBytes int64
}

func defaultFilterSettings() filterSettings {
	return filterSettings{
		shallowETag: shallowETagSettings{maxBodyBytes: DefaultShallowETagMaxBodyBytes},
	}
}

// WithCORS 设置并启用 CORS 过滤器。
func WithCORS(config gowebcors.Config) Option {
	return func(settings *settings) error {
		settings.filters.cors.config = config
		settings.filters.cors.enabled = true
		settings.filters.cors.enabledSet = true
		return nil
	}
}

// WithCORSEnabled 设置是否启用 CORS 过滤器。
func WithCORSEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.cors.enabled = enabled
		settings.filters.cors.enabledSet = true
		return nil
	}
}

// WithForwardedHeadersEnabled 设置是否启用代理头过滤器。
func WithForwardedHeadersEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.forwardedHeaders.enabled = enabled
		return nil
	}
}

// WithShallowETagEnabled 设置是否启用浅 ETag 过滤器。
func WithShallowETagEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.shallowETag.enabled = enabled
		return nil
	}
}

// WithShallowETagMaxBodyBytes 设置浅 ETag 最大缓存体积。
func WithShallowETagMaxBodyBytes(size int64) Option {
	return func(settings *settings) error {
		if size < 0 {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "shallow etag max body bytes %d must be >= 0", size)
		}
		settings.filters.shallowETag.maxBodyBytes = size
		return nil
	}
}

func (s *settings) applyFilterEnvironment(environment coreenv.Environment) error {
	if value, ok := firstProperty(environment, PropertyCORSEnabled, propertySpringWebCORSEnabled, propertySpringMVCCORSEnabled); ok {
		enabled, err := parseBoolProperty(PropertyCORSEnabled, value)
		if err != nil {
			return err
		}
		s.filters.cors.enabled = enabled
		s.filters.cors.enabledSet = true
	}
	if value, ok := firstProperty(environment, PropertyCORSAllowedOrigins, propertySpringWebCORSAllowedOrigins, propertySpringMVCCORSAllowedOrigins); ok {
		s.filters.cors.config.AllowedOrigins = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := firstProperty(environment, PropertyCORSAllowedOriginPatterns, propertySpringWebCORSAllowedOriginPatterns, propertySpringMVCCORSAllowedOriginPatterns); ok {
		s.filters.cors.config.AllowedOriginPatterns = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := firstProperty(environment, PropertyCORSAllowedMethods, propertySpringWebCORSAllowedMethods, propertySpringMVCCORSAllowedMethods); ok {
		s.filters.cors.config.AllowedMethods = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := firstProperty(environment, PropertyCORSAllowedHeaders, propertySpringWebCORSAllowedHeaders, propertySpringMVCCORSAllowedHeaders); ok {
		s.filters.cors.config.AllowedHeaders = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := firstProperty(environment, PropertyCORSExposedHeaders, propertySpringWebCORSExposedHeaders, propertySpringMVCCORSExposedHeaders); ok {
		s.filters.cors.config.ExposedHeaders = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := firstProperty(environment, PropertyCORSAllowCredentials, propertySpringWebCORSAllowCredentials, propertySpringMVCCORSAllowCredentials); ok {
		enabled, err := parseBoolProperty(PropertyCORSAllowCredentials, value)
		if err != nil {
			return err
		}
		s.filters.cors.config.AllowCredentials = enabled
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := firstProperty(environment, PropertyCORSMaxAge, propertySpringWebCORSMaxAge, propertySpringMVCCORSMaxAge); ok {
		maxAge, err := parseDurationProperty(PropertyCORSMaxAge, value)
		if err != nil {
			return err
		}
		s.filters.cors.config.MaxAge = maxAge
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyForwardedHeadersEnabled); ok {
		enabled, err := parseBoolProperty(PropertyForwardedHeadersEnabled, value)
		if err != nil {
			return err
		}
		s.filters.forwardedHeaders.enabled = enabled
	} else if value, ok := environment.GetProperty(propertySpringForwardedHeadersEnabled); ok {
		s.filters.forwardedHeaders.enabled = strings.EqualFold(strings.TrimSpace(value), "framework")
	}
	if value, ok := environment.GetProperty(PropertyShallowETagEnabled); ok {
		enabled, err := parseBoolProperty(PropertyShallowETagEnabled, value)
		if err != nil {
			return err
		}
		s.filters.shallowETag.enabled = enabled
	} else if value, ok := environment.GetProperty(propertySpringShallowETagEnabled); ok {
		enabled, err := parseBoolProperty(propertySpringShallowETagEnabled, value)
		if err != nil {
			return err
		}
		s.filters.shallowETag.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyShallowETagMaxBodyBytes); ok {
		size, err := parseInt64Property(PropertyShallowETagMaxBodyBytes, value)
		if err != nil {
			return err
		}
		s.filters.shallowETag.maxBodyBytes = size
	} else if value, ok := environment.GetProperty(propertySpringShallowETagMaxBodyBytes); ok {
		size, err := parseInt64Property(propertySpringShallowETagMaxBodyBytes, value)
		if err != nil {
			return err
		}
		s.filters.shallowETag.maxBodyBytes = size
	}
	return nil
}

func registerWebFilters(registry *goarkcontainer.Registry, settings filterSettings) error {
	if settings.forwardedHeaders.enabled {
		if err := goweb.RegisterFilter(registry, BeanNameForwardedHeadersFilter, gowebfilter.ForwardedHeaders(), goarkcontainer.WithOrder(orderForwardedHeadersFilter)); err != nil {
			return err
		}
	}
	if settings.cors.enabled {
		filter, err := gowebcors.New(settings.cors.config)
		if err != nil {
			return err
		}
		if err := goweb.RegisterFilter(registry, BeanNameCORSFilter, filter, goarkcontainer.WithOrder(orderCORSFilter)); err != nil {
			return err
		}
	}
	if settings.shallowETag.enabled {
		if err := goweb.RegisterFilter(registry, BeanNameShallowETagFilter, gowebfilter.ShallowETag(
			gowebfilter.WithMaxBodyBytes(settings.shallowETag.maxBodyBytes),
		), goarkcontainer.WithOrder(orderShallowETagFilter)); err != nil {
			return err
		}
	}
	return nil
}

func enableCORSByConfiguration(settings *corsFilterSettings) {
	if !settings.enabledSet {
		settings.enabled = true
	}
}

func firstProperty(environment coreenv.Environment, names ...string) (string, bool) {
	for _, name := range names {
		if value, ok := environment.GetProperty(name); ok {
			return value, true
		}
	}
	return "", false
}

func parseBoolProperty(name string, value string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid boolean property %s=%q", name, value)
	}
	return parsed, nil
}

func parseDurationProperty(name string, value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid duration property %s=%q", name, value)
	}
	return parsed, nil
}

func parseInt64Property(name string, value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid int64 property %s=%q", name, value)
	}
	if parsed < 0 {
		return 0, arkerrors.Newf(arkerrors.CodeInvalidArgument, "property %s=%q must be >= 0", name, value)
	}
	return parsed, nil
}
