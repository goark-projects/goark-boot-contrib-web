package gbcweb

import "time"

const (
	// BeanNameDeployment 是默认 Arkarta Servlet 部署 Bean 的稳定名称。
	BeanNameDeployment = "goark.boot.web.deployment"
	// BeanNameStaticResources 是默认静态资源配置器 Bean 名称。
	BeanNameStaticResources = "goark.boot.web.staticResources"
	// BeanNameCORSFilter 是默认 CORS 过滤器 Bean 名称。
	BeanNameCORSFilter = "goark.boot.web.corsFilter"
	// BeanNameForwardedHeadersFilter 是默认代理头过滤器 Bean 名称。
	BeanNameForwardedHeadersFilter = "goark.boot.web.forwardedHeadersFilter"
	// BeanNameShallowETagFilter 是默认浅 ETag 过滤器 Bean 名称。
	BeanNameShallowETagFilter = "goark.boot.web.shallowETagFilter"
	// BeanNameProblemDetailsMapper 是默认 Problem Details 错误映射器 Bean 名称。
	BeanNameProblemDetailsMapper = "goark.boot.web.problemDetailsMapper"
	// BeanNameErrorEndpoint 是默认错误端点配置器 Bean 名称。
	BeanNameErrorEndpoint = "goark.boot.web.errorEndpoint"
	// BeanNameViewResolver 是默认 MVC 视图解析器 Bean 名称。
	BeanNameViewResolver = "goark.boot.web.viewResolver"
	// BeanNameViewInterceptor 是默认 MVC 视图解析器拦截器 Bean 名称。
	BeanNameViewInterceptor = "goark.boot.web.viewInterceptor"
	// BeanNameHTTPClientBuilder 是默认 Web HTTP 客户端构建器 Bean 名称。
	BeanNameHTTPClientBuilder = "goark.boot.web.clientBuilder"
	// BeanNameHTTPClient 是默认 Web HTTP 客户端 Bean 名称。
	BeanNameHTTPClient = "goark.boot.web.client"
)

const (
	// DefaultApplicationName 是默认 Web 应用名称。
	DefaultApplicationName = "goark"
	// DefaultContextPath 是默认 Servlet 上下文路径。
	DefaultContextPath = "/"
	// DefaultMappingPattern 是默认 Servlet 映射模式。
	DefaultMappingPattern = "/"
	// DefaultStaticResourcesEnabled 表示默认启用静态资源约定。
	DefaultStaticResourcesEnabled = true
	// DefaultStaticResourcesLocation 是默认静态资源目录。
	DefaultStaticResourcesLocation = "resource/static"
	// DefaultStaticResourcesLocations 是默认静态资源目录列表，顺序对齐 Spring Boot 的常用资源约定。
	DefaultStaticResourcesLocations = "resource/static,resource/public,resource/resources,resource/META-INF/resources"
	// DefaultStaticResourcesPattern 是默认静态资源 Servlet 映射。
	DefaultStaticResourcesPattern = "/static/*"
	// DefaultStaticResourcesServletName 是默认静态资源 Servlet 名称。
	DefaultStaticResourcesServletName = "goark.boot.web.static"
	// DefaultShallowETagMaxBodyBytes 是浅 ETag 默认最大缓存体积。
	DefaultShallowETagMaxBodyBytes int64 = 1 << 20
	// DefaultErrorPath 是默认 Boot 风格错误端点路径。
	DefaultErrorPath = "/error"
	// DefaultViewTemplatesEnabled 表示默认启用模板视图约定。
	DefaultViewTemplatesEnabled = true
	// DefaultViewTemplatesLocation 是默认模板目录。
	DefaultViewTemplatesLocation = "resource/templates"
	// DefaultViewTemplateSuffix 是默认模板后缀。
	DefaultViewTemplateSuffix = ".html"
	// DefaultViewTemplateContentType 是默认模板响应媒体类型。
	DefaultViewTemplateContentType = "text/html; charset=utf-8"
	// DefaultHTTPClientEnabled 表示默认注册 Web HTTP 客户端 Bean。
	DefaultHTTPClientEnabled = true
	// DefaultHTTPClientTimeout 是默认 Web HTTP 客户端超时。
	DefaultHTTPClientTimeout = 30 * time.Second
	// DefaultHTTPClientMaxResponseBytes 是默认响应体快照最大读取字节数。
	DefaultHTTPClientMaxResponseBytes int64 = 16 << 20
)

const (
	// PropertyApplicationName 设置 Web 应用名称。
	PropertyApplicationName = "goark.web.application.name"
	// PropertyServletContextPath 设置 Servlet 上下文路径。
	PropertyServletContextPath = "goark.web.servlet.context-path"
	// PropertyServletMapping 设置 Servlet 映射模式。
	PropertyServletMapping = "goark.web.servlet.mapping"
	// PropertyStaticResourcesEnabled 设置是否启用静态资源约定。
	PropertyStaticResourcesEnabled = "goark.web.resources.static.enabled"
	// PropertyStaticResourcesLocations 设置静态资源目录列表。
	PropertyStaticResourcesLocations = "goark.web.resources.static.locations"
	// PropertyStaticResourcesLocation 设置单个静态资源目录。
	PropertyStaticResourcesLocation = "goark.web.resources.static.location"
	// PropertyStaticResourcesPattern 设置静态资源 Servlet 映射。
	PropertyStaticResourcesPattern = "goark.web.resources.static.pattern"
	// PropertyStaticResourcesServletName 设置静态资源 Servlet 名称。
	PropertyStaticResourcesServletName = "goark.web.resources.static.servlet-name"
	// PropertyStaticResourcesWelcomeFiles 设置静态资源 welcome file 列表。
	PropertyStaticResourcesWelcomeFiles = "goark.web.resources.static.welcome-files"
	// PropertyStaticResourcesCacheControl 设置静态资源 Cache-Control 头。
	PropertyStaticResourcesCacheControl = "goark.web.resources.static.cache-control"
	// PropertyStaticResourcesCacheMaxAge 设置静态资源 public max-age 缓存时间。
	PropertyStaticResourcesCacheMaxAge = "goark.web.resources.static.cache.max-age"
	// PropertyCORSEnabled 设置是否启用 CORS 过滤器。
	PropertyCORSEnabled = "goark.web.cors.enabled"
	// PropertyCORSAllowedOrigins 设置允许的 CORS Origin 列表。
	PropertyCORSAllowedOrigins = "goark.web.cors.allowed-origins"
	// PropertyCORSAllowedOriginPatterns 设置允许的 CORS Origin 通配规则列表。
	PropertyCORSAllowedOriginPatterns = "goark.web.cors.allowed-origin-patterns"
	// PropertyCORSAllowedMethods 设置允许的 CORS HTTP 方法列表。
	PropertyCORSAllowedMethods = "goark.web.cors.allowed-methods"
	// PropertyCORSAllowedHeaders 设置允许的 CORS 请求头列表。
	PropertyCORSAllowedHeaders = "goark.web.cors.allowed-headers"
	// PropertyCORSExposedHeaders 设置暴露给浏览器的 CORS 响应头列表。
	PropertyCORSExposedHeaders = "goark.web.cors.exposed-headers"
	// PropertyCORSAllowCredentials 设置是否允许 CORS 携带凭据。
	PropertyCORSAllowCredentials = "goark.web.cors.allow-credentials"
	// PropertyCORSMaxAge 设置 CORS 预检结果缓存时间。
	PropertyCORSMaxAge = "goark.web.cors.max-age"
	// PropertyForwardedHeadersEnabled 设置是否启用代理头过滤器。
	PropertyForwardedHeadersEnabled = "goark.web.filters.forwarded-headers.enabled"
	// PropertyShallowETagEnabled 设置是否启用浅 ETag 过滤器。
	PropertyShallowETagEnabled = "goark.web.filters.shallow-etag.enabled"
	// PropertyShallowETagMaxBodyBytes 设置浅 ETag 最大缓存体积。
	PropertyShallowETagMaxBodyBytes = "goark.web.filters.shallow-etag.max-body-bytes"
	// PropertyErrorEndpointEnabled 设置是否注册默认错误端点。
	PropertyErrorEndpointEnabled = "goark.web.error.enabled"
	// PropertyErrorPath 设置默认错误端点路径。
	PropertyErrorPath = "goark.web.error.path"
	// PropertyProblemDetailsEnabled 设置是否启用 Problem Details 错误响应。
	PropertyProblemDetailsEnabled = "goark.web.problem-details.enabled"
	// PropertyViewTemplatesEnabled 设置是否启用 MVC 模板视图约定。
	PropertyViewTemplatesEnabled = "goark.web.mvc.view.enabled"
	// PropertyViewTemplatesLocation 设置 MVC 模板根目录。
	PropertyViewTemplatesLocation = "goark.web.mvc.view.templates.location"
	// PropertyViewTemplatePrefix 设置 MVC 模板逻辑视图名前缀。
	PropertyViewTemplatePrefix = "goark.web.mvc.view.prefix"
	// PropertyViewTemplateSuffix 设置 MVC 模板逻辑视图名后缀。
	PropertyViewTemplateSuffix = "goark.web.mvc.view.suffix"
	// PropertyViewTemplateContentType 设置 MVC 模板响应媒体类型。
	PropertyViewTemplateContentType = "goark.web.mvc.view.content-type"
	// PropertyHTTPClientEnabled 设置是否注册默认 Web HTTP 客户端。
	PropertyHTTPClientEnabled = "goark.web.client.enabled"
	// PropertyHTTPClientBaseURL 设置默认 Web HTTP 客户端基础 URL。
	PropertyHTTPClientBaseURL = "goark.web.client.base-url"
	// PropertyHTTPClientTimeout 设置默认 Web HTTP 客户端超时。
	PropertyHTTPClientTimeout = "goark.web.client.timeout"
	// PropertyHTTPClientMaxResponseBytes 设置默认 Web HTTP 客户端响应体快照上限。
	PropertyHTTPClientMaxResponseBytes = "goark.web.client.max-response-bytes"
	// PropertyHTTPClientDefaultHeaders 设置默认 Web HTTP 客户端请求头，格式为 Name=Value,Name2=Value2。
	PropertyHTTPClientDefaultHeaders = "goark.web.client.default-headers"
)
