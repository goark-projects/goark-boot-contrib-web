package gbcweb

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	goarkcontainer "goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/problem"
)

const (
	propertyServerErrorPath   = "server.error.path"
	orderProblemDetailsMapper = 10000
	orderErrorEndpoint        = 10010
)

type errorHandlingSettings struct {
	enabled               bool
	problemDetailsEnabled bool
	path                  string
}

type errorEndpointConfigurer struct {
	path string
}

func defaultErrorHandlingSettings() errorHandlingSettings {
	return errorHandlingSettings{
		enabled:               true,
		problemDetailsEnabled: true,
		path:                  DefaultErrorPath,
	}
}

// WithErrorEndpointEnabled 设置是否注册默认错误端点。
func WithErrorEndpointEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.errorHandling.enabled = enabled
		return nil
	}
}

// WithErrorPath 设置默认错误端点路径。
func WithErrorPath(path string) Option {
	return func(settings *settings) error {
		path = strings.TrimSpace(path)
		if path == "" || !strings.HasPrefix(path, "/") {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "web error path %q must start with /", path)
		}
		settings.errorHandling.path = path
		return nil
	}
}

// WithProblemDetailsEnabled 设置是否启用 Problem Details 错误响应。
func WithProblemDetailsEnabled(enabled bool) Option {
	return func(settings *settings) error {
		settings.errorHandling.problemDetailsEnabled = enabled
		return nil
	}
}

func (s *settings) applyErrorEnvironment(environment coreenv.Environment) error {
	if value, ok := environment.GetProperty(PropertyErrorEndpointEnabled); ok {
		enabled, err := parseBoolProperty(PropertyErrorEndpointEnabled, value)
		if err != nil {
			return err
		}
		s.errorHandling.enabled = enabled
	}
	if value, ok := firstProperty(environment, PropertyErrorPath, propertyServerErrorPath); ok {
		if err := WithErrorPath(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyProblemDetailsEnabled); ok {
		enabled, err := parseBoolProperty(PropertyProblemDetailsEnabled, value)
		if err != nil {
			return err
		}
		s.errorHandling.problemDetailsEnabled = enabled
	}
	return nil
}

func registerErrorHandling(registry *goarkcontainer.Registry, settings errorHandlingSettings) error {
	if settings.problemDetailsEnabled {
		if err := goweb.RegisterFallbackErrorMapper(registry, BeanNameProblemDetailsMapper, problem.NewMapper(), goarkcontainer.WithOrder(orderProblemDetailsMapper)); err != nil {
			return err
		}
	}
	if !settings.enabled {
		return nil
	}
	return goweb.RegisterConfigurer(registry, BeanNameErrorEndpoint, errorEndpointConfigurer{path: settings.path}, goarkcontainer.WithOrder(orderErrorEndpoint))
}

func (c errorEndpointConfigurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	for _, method := range errorEndpointMethods() {
		if routeExists(registry, method, c.path) {
			continue
		}
		if err := registry.Handle(method, c.path, arkweb.HandlerFunc(errorEndpointHandler)); err != nil {
			return err
		}
	}
	return nil
}

func errorEndpointMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}
}

func routeExists(registry *goweb.Registry, method string, path string) bool {
	for _, route := range registry.Routes() {
		if route.Method == method && route.Pattern == path {
			return true
		}
	}
	return false
}

func errorEndpointHandler(ctx *arkweb.Context) (arkweb.Result, error) {
	statusCode := errorStatusCode(ctx)
	return problem.New(statusCode,
		problem.WithDetail(errorMessage(ctx, statusCode)),
		problem.WithInstance(errorInstance(ctx)),
		problem.WithExtension("error", fmt.Sprintf("HTTP_%d", statusCode)),
	), nil
}

func errorStatusCode(ctx *arkweb.Context) int {
	value, ok := requestAttribute(ctx, servlet.AttributeErrorStatusCode)
	if !ok {
		return http.StatusInternalServerError
	}
	switch typed := value.(type) {
	case int:
		return normalizeHTTPStatus(typed)
	case int64:
		return normalizeHTTPStatus(int(typed))
	case string:
		return normalizeHTTPStatusString(typed)
	default:
		return http.StatusInternalServerError
	}
}

func errorMessage(ctx *arkweb.Context, statusCode int) string {
	if value, ok := requestAttribute(ctx, servlet.AttributeErrorMessage); ok {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	if text := http.StatusText(statusCode); text != "" {
		return text
	}
	return "HTTP error"
}

func errorInstance(ctx *arkweb.Context) string {
	if value, ok := requestAttribute(ctx, servlet.AttributeErrorRequestURI); ok {
		if path, ok := value.(string); ok && strings.TrimSpace(path) != "" {
			return path
		}
	}
	if ctx != nil && ctx.Request() != nil {
		return ctx.Request().Path()
	}
	return ""
}

func requestAttribute(ctx *arkweb.Context, name string) (any, bool) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false
	}
	return ctx.Request().Attribute(name)
}

func normalizeHTTPStatus(statusCode int) int {
	if statusCode < 100 || statusCode > 999 {
		return http.StatusInternalServerError
	}
	return statusCode
}

func normalizeHTTPStatusString(value string) int {
	statusCode, err := strconvAtoi(strings.TrimSpace(value))
	if err != nil {
		return http.StatusInternalServerError
	}
	return normalizeHTTPStatus(statusCode)
}

func strconvAtoi(value string) (int, error) {
	var out int
	if value == "" {
		return 0, arkerrors.New(arkerrors.CodeInvalidArgument, "empty status code")
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c < '0' || c > '9' {
			return 0, arkerrors.Newf(arkerrors.CodeInvalidArgument, "invalid status code %q", value)
		}
		out = out*10 + int(c-'0')
	}
	return out, nil
}
