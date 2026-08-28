package gbcweb

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
	// DefaultStaticResourcesPattern 是默认静态资源 Servlet 映射。
	DefaultStaticResourcesPattern = "/static/*"
	// DefaultStaticResourcesServletName 是默认静态资源 Servlet 名称。
	DefaultStaticResourcesServletName = "goark.boot.web.static"
	// DefaultShallowETagMaxBodyBytes 是浅 ETag 默认最大缓存体积。
	DefaultShallowETagMaxBodyBytes int64 = 1 << 20
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
)
