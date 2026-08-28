package gbcweb

import (
	"strconv"
	"strings"
	"time"

	"goark.dev/arkarta/servlet"
	servletcontainer "goark.dev/arkarta/servlet/container"
	arkweb "goark.dev/arkarta/web"
	gbcarkhos "goark.dev/gbc-arkhos"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
)

// Option 定制 Web 自动配置。
type Option func(*settings) error

type settings struct {
	applicationName   string
	contextPath       string
	mappingPattern    string
	routerOptions     []arkweb.Option
	webAppOptions     []servlet.WebAppOption
	deploymentOptions []servletcontainer.DeploymentOption
	arkhosOptions     []gbcarkhos.Option
	staticResources   staticResourceSettings
	viewTemplates     viewTemplateSettings
	filters           filterSettings
	errorHandling     errorHandlingSettings
}

// WithApplicationName 设置 Web 应用名称。
func WithApplicationName(name string) Option {
	return func(config *settings) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "web application name is empty")
		}
		config.applicationName = name
		return nil
	}
}

// WithContextPath 设置 Servlet 上下文路径。
func WithContextPath(path string) Option {
	return func(config *settings) error {
		path = strings.TrimSpace(path)
		if path == "" {
			path = DefaultContextPath
		}
		if !strings.HasPrefix(path, "/") {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "web context path %q must start with /", path)
		}
		config.contextPath = path
		return nil
	}
}

// WithMappingPattern 设置 Servlet 映射模式。
func WithMappingPattern(pattern string) Option {
	return func(config *settings) error {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			pattern = DefaultMappingPattern
		}
		config.mappingPattern = pattern
		return nil
	}
}

// WithStaticResourcesEnabled 设置是否启用默认静态资源约定。
func WithStaticResourcesEnabled(enabled bool) Option {
	return func(config *settings) error {
		config.staticResources.enabled = enabled
		return nil
	}
}

// WithStaticResourceLocations 设置默认静态资源目录。
func WithStaticResourceLocations(locations ...string) Option {
	copied := append([]string(nil), locations...)
	return func(config *settings) error {
		normalized, err := normalizeStaticResourceLocations(copied)
		if err != nil {
			return err
		}
		config.staticResources.locations = normalized
		return nil
	}
}

// WithStaticResourcePattern 设置默认静态资源 Servlet 映射。
func WithStaticResourcePattern(pattern string) Option {
	return func(config *settings) error {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "static resource pattern is empty")
		}
		config.staticResources.pattern = pattern
		return nil
	}
}

// WithStaticResourceServletName 设置默认静态资源 Servlet 名称。
func WithStaticResourceServletName(name string) Option {
	return func(config *settings) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "static resource servlet name is empty")
		}
		config.staticResources.servletName = name
		return nil
	}
}

// WithStaticResourceWelcomeFiles 设置默认静态资源 welcome file。
func WithStaticResourceWelcomeFiles(files ...string) Option {
	copied := append([]string(nil), files...)
	return func(config *settings) error {
		config.staticResources.welcomeFiles = splitStaticResourceList(copied)
		config.staticResources.welcomeFilesSet = true
		return nil
	}
}

// WithStaticResourceCacheControl 设置默认静态资源 Cache-Control 头。
func WithStaticResourceCacheControl(value string) Option {
	return func(config *settings) error {
		value = strings.TrimSpace(value)
		if strings.ContainsAny(value, "\r\n") {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "static resource cache-control is invalid")
		}
		config.staticResources.cacheControl = value
		return nil
	}
}

// WithStaticResourceCacheMaxAge 设置默认静态资源 public max-age 缓存时间。
func WithStaticResourceCacheMaxAge(maxAge time.Duration) Option {
	return func(config *settings) error {
		if maxAge < 0 {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "static resource cache max-age %s must be >= 0", maxAge)
		}
		config.staticResources.cacheControl = "public, max-age=" + strconv.FormatInt(int64(maxAge/time.Second), 10)
		return nil
	}
}

// WithViewTemplatesEnabled 设置是否启用默认 MVC 模板视图约定。
func WithViewTemplatesEnabled(enabled bool) Option {
	return func(config *settings) error {
		config.viewTemplates.enabled = enabled
		return nil
	}
}

// WithViewTemplatesLocation 设置默认 MVC 模板根目录。
func WithViewTemplatesLocation(location string) Option {
	return func(config *settings) error {
		location = normalizeStaticResourceLocation(location)
		if strings.TrimSpace(location) == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "view template location is empty")
		}
		config.viewTemplates.location = location
		return nil
	}
}

// WithViewTemplatePrefix 设置默认 MVC 模板逻辑视图名前缀。
func WithViewTemplatePrefix(prefix string) Option {
	return func(config *settings) error {
		config.viewTemplates.prefix = strings.TrimSpace(prefix)
		return nil
	}
}

// WithViewTemplateSuffix 设置默认 MVC 模板逻辑视图名后缀。
func WithViewTemplateSuffix(suffix string) Option {
	return func(config *settings) error {
		config.viewTemplates.suffix = strings.TrimSpace(suffix)
		return nil
	}
}

// WithViewTemplateContentType 设置默认 MVC 模板响应媒体类型。
func WithViewTemplateContentType(contentType string) Option {
	return func(config *settings) error {
		contentType = strings.TrimSpace(contentType)
		if strings.ContainsAny(contentType, "\r\n") {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "view template content type is invalid")
		}
		if contentType != "" {
			config.viewTemplates.contentType = contentType
		}
		return nil
	}
}

// WithRouterOptions 追加 Arkarta Web Router 选项。
func WithRouterOptions(options ...arkweb.Option) Option {
	copied := append([]arkweb.Option(nil), options...)
	return func(config *settings) error {
		for _, option := range copied {
			if option != nil {
				config.routerOptions = append(config.routerOptions, option)
			}
		}
		return nil
	}
}

// WithWebAppOptions 追加 Arkarta Servlet WebApp 选项。
func WithWebAppOptions(options ...servlet.WebAppOption) Option {
	copied := append([]servlet.WebAppOption(nil), options...)
	return func(config *settings) error {
		for _, option := range copied {
			if option != nil {
				config.webAppOptions = append(config.webAppOptions, option)
			}
		}
		return nil
	}
}

// WithDeploymentOptions 追加 Arkarta Servlet 部署选项。
func WithDeploymentOptions(options ...servletcontainer.DeploymentOption) Option {
	copied := append([]servletcontainer.DeploymentOption(nil), options...)
	return func(config *settings) error {
		for _, option := range copied {
			if option != nil {
				config.deploymentOptions = append(config.deploymentOptions, option)
			}
		}
		return nil
	}
}

// WithArkhosOptions 透传底层 Arkhos starter 选项。
func WithArkhosOptions(options ...gbcarkhos.Option) Option {
	copied := append([]gbcarkhos.Option(nil), options...)
	return func(config *settings) error {
		for _, option := range copied {
			if option != nil {
				config.arkhosOptions = append(config.arkhosOptions, option)
			}
		}
		return nil
	}
}

func newSettings(environment coreenv.Environment, options []Option) (settings, error) {
	config := settings{
		applicationName: DefaultApplicationName,
		contextPath:     DefaultContextPath,
		mappingPattern:  DefaultMappingPattern,
		staticResources: defaultStaticResourceSettings(),
		viewTemplates:   defaultViewTemplateSettings(),
		filters:         defaultFilterSettings(),
		errorHandling:   defaultErrorHandlingSettings(),
	}
	if err := config.applyEnvironment(environment); err != nil {
		return settings{}, err
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return settings{}, err
		}
	}
	return config, nil
}

func (s *settings) applyEnvironment(environment coreenv.Environment) error {
	if environment == nil {
		return nil
	}
	if value, ok := environment.GetProperty(PropertyApplicationName); ok {
		if err := WithApplicationName(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyServletContextPath); ok {
		if err := WithContextPath(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyServletMapping); ok {
		if err := WithMappingPattern(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesEnabled); ok {
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid static resources enabled value %q", value)
		}
		if err := WithStaticResourcesEnabled(enabled)(s); err != nil {
			return err
		}
	}
	if value, ok := staticResourceLocationsProperty(environment); ok {
		if err := WithStaticResourceLocations(value)(s); err != nil {
			return err
		}
	}
	if value, ok := staticResourcePatternProperty(environment); ok {
		if err := WithStaticResourcePattern(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesServletName); ok {
		if err := WithStaticResourceServletName(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesWelcomeFiles); ok {
		if err := WithStaticResourceWelcomeFiles(value)(s); err != nil {
			return err
		}
	}
	if value, ok := firstProperty(environment, PropertyStaticResourcesCacheControl, propertySpringStaticCacheControl); ok {
		if err := WithStaticResourceCacheControl(value)(s); err != nil {
			return err
		}
	} else if value, ok := firstProperty(environment, PropertyStaticResourcesCacheMaxAge, propertySpringStaticCacheMaxAge, propertySpringStaticCachePeriod); ok {
		maxAge, err := parseDurationProperty(PropertyStaticResourcesCacheMaxAge, value)
		if err != nil {
			return err
		}
		if err := WithStaticResourceCacheMaxAge(maxAge)(s); err != nil {
			return err
		}
	}
	if err := s.applyFilterEnvironment(environment); err != nil {
		return err
	}
	if err := s.applyViewEnvironment(environment); err != nil {
		return err
	}
	if err := s.applyErrorEnvironment(environment); err != nil {
		return err
	}
	return nil
}
