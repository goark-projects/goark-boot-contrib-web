package gbcweb

import (
	"strings"
	"time"

	arkerrors "goark.dev/goark/errors"
	gowebcors "goark.dev/goark/web/cors"
	gowebfilter "goark.dev/goark/web/filter"
)

const (
	propertyServerForwardHeadersStrategy = "server.forward-headers-strategy"
	orderCharacterEncodingFilter         = -400
	orderForwardedHeadersFilter          = -300
	orderCORSFilter                      = -200
	orderHiddenHTTPMethodFilter          = -100
	orderFormContentFilter               = -90
	orderFlashMapFilter                  = -80
	orderSessionAttributesFilter         = -70
	orderShallowETagFilter               = 100
)

type filterSettings struct {
	characterEncoding characterEncodingSettings
	cors              corsFilterSettings
	forwardedHeaders  forwardedHeadersSettings
	formContent       formContentSettings
	flashMap          flashMapSettings
	hiddenMethod      hiddenMethodSettings
	sessionAttributes sessionAttributesSettings
	shallowETag       shallowETagSettings
}

type corsFilterSettings struct {
	enabled    bool
	enabledSet bool
	config     gowebcors.Config
}

type forwardedHeadersSettings struct {
	enabled bool
}

type characterEncodingSettings struct {
	enabled       bool
	encoding      string
	forceRequest  bool
	forceResponse bool
	options       []gowebfilter.CharacterEncodingOption
}

type shallowETagSettings struct {
	enabled      bool
	maxBodyBytes int64
}

type hiddenMethodSettings struct {
	enabled bool
	options []gowebfilter.HiddenMethodOption
}

type formContentSettings struct {
	enabled      bool
	maxBodyBytes int64
	options      []gowebfilter.FormContentOption
}

type flashMapSettings struct {
	enabled bool
	timeout time.Duration
}

type sessionAttributesSettings struct {
	enabled bool
}

func defaultFilterSettings() filterSettings {
	return filterSettings{
		characterEncoding: characterEncodingSettings{
			enabled:       DefaultCharacterEncodingFilterEnabled,
			encoding:      DefaultCharacterEncoding,
			forceRequest:  DefaultForceRequestCharacterEncoding,
			forceResponse: DefaultForceResponseCharacterEncoding,
		},
		formContent: formContentSettings{
			enabled:      DefaultFormContentFilterEnabled,
			maxBodyBytes: DefaultFormContentMaxBodyBytes,
		},
		flashMap: flashMapSettings{
			enabled: DefaultFlashMapFilterEnabled,
			timeout: DefaultFlashMapTimeout,
		},
		hiddenMethod: hiddenMethodSettings{
			enabled: DefaultHiddenHTTPMethodFilterEnabled,
		},
		sessionAttributes: sessionAttributesSettings{
			enabled: DefaultSessionAttributesFilterEnabled,
		},
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

// WithCharacterEncodingFilterEnabled 设置是否启用字符集过滤器。
func WithCharacterEncodingFilterEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.characterEncoding.enabled = enabled
		return nil
	}
}

// WithCharacterEncoding 设置 Web 默认字符集。
func WithCharacterEncoding(encoding string) Option {
	return func(settings *settings) error {
		encoding = strings.TrimSpace(encoding)
		if encoding != "" {
			settings.filters.characterEncoding.encoding = encoding
		}
		return nil
	}
}

// WithForceCharacterEncoding 设置是否同时强制请求和响应字符集。
func WithForceCharacterEncoding(force bool) Option {
	return func(settings *settings) error {
		settings.filters.characterEncoding.forceRequest = force
		settings.filters.characterEncoding.forceResponse = force
		return nil
	}
}

// WithForceRequestCharacterEncoding 设置是否强制请求字符集。
func WithForceRequestCharacterEncoding(force bool) Option {
	return func(settings *settings) error {
		settings.filters.characterEncoding.forceRequest = force
		return nil
	}
}

// WithForceResponseCharacterEncoding 设置是否强制响应字符集。
func WithForceResponseCharacterEncoding(force bool) Option {
	return func(settings *settings) error {
		settings.filters.characterEncoding.forceResponse = force
		return nil
	}
}

// WithCharacterEncodingFilterOptions 设置字符集过滤器选项并启用过滤器。
func WithCharacterEncodingFilterOptions(options ...gowebfilter.CharacterEncodingOption) Option {
	copied := append([]gowebfilter.CharacterEncodingOption(nil), options...)
	return func(settings *settings) error {
		settings.filters.characterEncoding.options = append(
			settings.filters.characterEncoding.options,
			copied...)
		settings.filters.characterEncoding.enabled = true
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
			return arkerrors.Newf(
				arkerrors.CodeInvalidArgument,
				"shallow etag max body bytes %d must be >= 0",
				size,
			)
		}
		settings.filters.shallowETag.maxBodyBytes = size
		return nil
	}
}

// WithHiddenHTTPMethodFilterEnabled 设置是否启用隐藏 HTTP 方法过滤器。
func WithHiddenHTTPMethodFilterEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.hiddenMethod.enabled = enabled
		return nil
	}
}

// WithHiddenHTTPMethodFilterOptions 设置隐藏 HTTP 方法过滤器选项并启用过滤器。
func WithHiddenHTTPMethodFilterOptions(options ...gowebfilter.HiddenMethodOption) Option {
	copied := append([]gowebfilter.HiddenMethodOption(nil), options...)
	return func(settings *settings) error {
		settings.filters.hiddenMethod.options = append(
			settings.filters.hiddenMethod.options,
			copied...)
		settings.filters.hiddenMethod.enabled = true
		return nil
	}
}

// WithFormContentFilterEnabled 设置是否启用表单内容过滤器。
func WithFormContentFilterEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.formContent.enabled = enabled
		return nil
	}
}

// WithFormContentMaxBodyBytes 设置表单内容过滤器最大缓存体积。
func WithFormContentMaxBodyBytes(size int64) Option {
	return func(settings *settings) error {
		if size < 0 {
			return arkerrors.Newf(
				arkerrors.CodeInvalidArgument,
				"form content max body bytes %d must be >= 0",
				size,
			)
		}
		settings.filters.formContent.maxBodyBytes = size
		return nil
	}
}

// WithFormContentFilterOptions 设置表单内容过滤器选项并启用过滤器。
func WithFormContentFilterOptions(options ...gowebfilter.FormContentOption) Option {
	copied := append([]gowebfilter.FormContentOption(nil), options...)
	return func(settings *settings) error {
		settings.filters.formContent.options = append(
			settings.filters.formContent.options,
			copied...)
		settings.filters.formContent.enabled = true
		return nil
	}
}

// WithFlashMapFilterEnabled 设置是否启用 MVC FlashMap 过滤器。
func WithFlashMapFilterEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.flashMap.enabled = enabled
		return nil
	}
}

// WithFlashMapTimeout 设置 MVC FlashMap 过期时间。
func WithFlashMapTimeout(timeout time.Duration) Option {
	return func(settings *settings) error {
		if timeout <= 0 {
			return arkerrors.Newf(
				arkerrors.CodeInvalidArgument,
				"flash map timeout %s must be > 0",
				timeout,
			)
		}
		settings.filters.flashMap.timeout = timeout
		return nil
	}
}

// WithSessionAttributesFilterEnabled 设置是否启用 MVC SessionAttributes 过滤器。
func WithSessionAttributesFilterEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.filters.sessionAttributes.enabled = enabled
		return nil
	}
}
