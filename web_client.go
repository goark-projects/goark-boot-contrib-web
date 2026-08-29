package gbcweb

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	goarkcontainer "goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	webclient "goark.dev/goark/web/client"
)

type httpClientSettings struct {
	enabled          bool
	baseURL          string
	timeout          time.Duration
	maxResponseBytes int64
	defaultHeaders   http.Header
	options          []webclient.Option
}

func defaultHTTPClientSettings() httpClientSettings {
	return httpClientSettings{
		enabled:          DefaultHTTPClientEnabled,
		timeout:          DefaultHTTPClientTimeout,
		maxResponseBytes: DefaultHTTPClientMaxResponseBytes,
		defaultHeaders:   make(http.Header),
	}
}

// WithHTTPClientEnabled 设置是否注册默认 Web HTTP 客户端。
func WithHTTPClientEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.httpClient.enabled = enabled
		return nil
	}
}

// WithHTTPClientBaseURL 设置默认 Web HTTP 客户端基础 URL。
func WithHTTPClientBaseURL(rawURL string) Option {
	return func(settings *settings) error {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL != "" {
			if err := validateHTTPClientBaseURL(rawURL); err != nil {
				return err
			}
		}
		settings.httpClient.baseURL = rawURL
		return nil
	}
}

// WithHTTPClientTimeout 设置默认 Web HTTP 客户端超时。
func WithHTTPClientTimeout(timeout time.Duration) Option {
	return func(settings *settings) error {
		if timeout < 0 {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "web http client timeout %s must be >= 0", timeout)
		}
		if timeout > 0 {
			settings.httpClient.timeout = timeout
		}
		return nil
	}
}

// WithHTTPClientMaxResponseBytes 设置默认 Web HTTP 客户端响应体快照上限，-1 表示不限制。
func WithHTTPClientMaxResponseBytes(size int64) Option {
	return func(settings *settings) error {
		if size < -1 {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "web http client max response bytes %d must be >= -1", size)
		}
		settings.httpClient.maxResponseBytes = size
		return nil
	}
}

// WithHTTPClientDefaultHeader 设置默认 Web HTTP 客户端请求头。
func WithHTTPClientDefaultHeader(name string, values ...string) Option {
	copied := append([]string(nil), values...)
	return func(settings *settings) error {
		if settings.httpClient.defaultHeaders == nil {
			settings.httpClient.defaultHeaders = make(http.Header)
		}
		return appendHTTPClientDefaultHeader(settings.httpClient.defaultHeaders, name, copied)
	}
}

// WithHTTPClientOptions 追加底层 Web HTTP 客户端选项。
func WithHTTPClientOptions(options ...webclient.Option) Option {
	copied := append([]webclient.Option(nil), options...)
	return func(settings *settings) error {
		for _, option := range copied {
			if option != nil {
				settings.httpClient.options = append(settings.httpClient.options, option)
			}
		}
		return nil
	}
}

func (s *settings) applyHTTPClientEnvironment(environment coreenv.Environment) error {
	if value, ok := environment.GetProperty(PropertyHTTPClientEnabled); ok {
		enabled, err := parseBoolProperty(PropertyHTTPClientEnabled, value)
		if err != nil {
			return err
		}
		s.httpClient.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyHTTPClientBaseURL); ok {
		if err := WithHTTPClientBaseURL(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyHTTPClientTimeout); ok {
		timeout, err := parseDurationProperty(PropertyHTTPClientTimeout, value)
		if err != nil {
			return err
		}
		if err := WithHTTPClientTimeout(timeout)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyHTTPClientMaxResponseBytes); ok {
		size, err := parseHTTPClientMaxResponseBytes(PropertyHTTPClientMaxResponseBytes, value)
		if err != nil {
			return err
		}
		if err := WithHTTPClientMaxResponseBytes(size)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyHTTPClientDefaultHeaders); ok {
		headers, err := parseHTTPClientDefaultHeaders(value)
		if err != nil {
			return err
		}
		s.httpClient.defaultHeaders = headers
	}
	return nil
}

func registerHTTPClient(registry *goarkcontainer.Registry, settings httpClientSettings) error {
	if !settings.enabled {
		return nil
	}
	options := settings.clientOptions()
	if _, exists := registry.Definition(BeanNameHTTPClientBuilder); !exists {
		copied := append([]webclient.Option(nil), options...)
		if err := goarkcontainer.Register[*webclient.Builder](registry, BeanNameHTTPClientBuilder, func(ctx context.Context, resolver goarkcontainer.Resolver) (*webclient.Builder, error) {
			return newHTTPClientBuilder(ctx, resolver, copied)
		}); err != nil {
			return err
		}
	}
	if _, exists := registry.Definition(BeanNameHTTPClient); exists {
		return nil
	}
	return goarkcontainer.Register[*webclient.Client](registry, BeanNameHTTPClient, func(ctx context.Context, resolver goarkcontainer.Resolver) (*webclient.Client, error) {
		builder, err := goarkcontainer.Get[*webclient.Builder](ctx, resolver, BeanNameHTTPClientBuilder)
		if err != nil {
			return nil, err
		}
		return builder.Build()
	})
}

func (s httpClientSettings) clientOptions() []webclient.Option {
	options := make([]webclient.Option, 0, 3+len(s.defaultHeaders)+len(s.options))
	if strings.TrimSpace(s.baseURL) != "" {
		options = append(options, webclient.WithBaseURL(s.baseURL))
	}
	if s.timeout > 0 {
		options = append(options, webclient.WithTimeout(s.timeout))
	}
	options = append(options, webclient.WithMaxResponseBytes(s.maxResponseBytes))
	for _, name := range sortedHeaderNames(s.defaultHeaders) {
		options = append(options, webclient.WithDefaultHeader(name, s.defaultHeaders.Values(name)...))
	}
	options = append(options, s.options...)
	return options
}

func validateHTTPClientBaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return arkerrors.Newf(arkerrors.CodeInvalidArgument, "web http client base url %q is invalid", rawURL)
	}
	return nil
}

func parseHTTPClientDefaultHeaders(raw string) (http.Header, error) {
	headers := make(http.Header)
	for _, item := range splitStaticResourceList([]string{raw}) {
		name, value, ok := strings.Cut(item, "=")
		if !ok {
			return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "web http client default header %q must use Name=Value", item)
		}
		if err := appendHTTPClientDefaultHeader(headers, name, []string{strings.TrimSpace(value)}); err != nil {
			return nil, err
		}
	}
	return headers, nil
}

func appendHTTPClientDefaultHeader(headers http.Header, name string, values []string) error {
	name = http.CanonicalHeaderKey(strings.TrimSpace(name))
	if name == "" || len(values) == 0 || strings.ContainsAny(name, "\r\n") {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "web http client default header is invalid")
	}
	for _, value := range values {
		if strings.ContainsAny(value, "\r\n") {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "web http client default header is invalid")
		}
		headers.Add(name, value)
	}
	return nil
}

func parseHTTPClientMaxResponseBytes(name string, value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid int64 property %s=%q", name, value)
	}
	if parsed < -1 {
		return 0, arkerrors.Newf(arkerrors.CodeInvalidArgument, "property %s=%q must be >= -1", name, value)
	}
	return parsed, nil
}

func sortedHeaderNames(headers http.Header) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
