package gbcweb

const (
	// BeanNameDeployment 是默认 Arkarta Servlet 部署 Bean 的稳定名称。
	BeanNameDeployment = "goark.boot.web.deployment"
	// BeanNameStaticResources 是默认静态资源配置器 Bean 名称。
	BeanNameStaticResources = "goark.boot.web.staticResources"
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
)
