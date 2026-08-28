package gbcweb

import (
	"context"
	"testing"
	"time"

	gbcarkhos "goark.dev/gbc-arkhos"
	goarkcontainer "goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	webclient "goark.dev/goark/web/client"
	gowebcors "goark.dev/goark/web/cors"
)

func TestNewSettings_whenEnvironmentIsNil_shouldUseWebDefaults(t *testing.T) {
	settings, err := newSettings(nil, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if settings.applicationName != DefaultApplicationName {
		t.Fatalf("application name = %q, want %q", settings.applicationName, DefaultApplicationName)
	}
	if settings.contextPath != DefaultContextPath {
		t.Fatalf("context path = %q, want %q", settings.contextPath, DefaultContextPath)
	}
	if settings.mappingPattern != DefaultMappingPattern {
		t.Fatalf("mapping pattern = %q, want %q", settings.mappingPattern, DefaultMappingPattern)
	}
	defaultLocations := splitStaticResourceList([]string{DefaultStaticResourcesLocations})
	if !settings.staticResources.enabled ||
		len(settings.staticResources.locations) != len(defaultLocations) ||
		settings.staticResources.locations[0] != "resource/static" ||
		settings.staticResources.locations[len(settings.staticResources.locations)-1] != "resource/META-INF/resources" ||
		settings.staticResources.pattern != DefaultStaticResourcesPattern {
		t.Fatalf("static resources defaults = %+v", settings.staticResources)
	}
	if settings.filters.cors.enabled ||
		settings.filters.forwardedHeaders.enabled ||
		settings.filters.shallowETag.enabled {
		t.Fatalf("filter defaults = %+v", settings.filters)
	}
	if !settings.viewTemplates.enabled ||
		settings.viewTemplates.location != DefaultViewTemplatesLocation ||
		settings.viewTemplates.suffix != DefaultViewTemplateSuffix ||
		settings.viewTemplates.contentType != DefaultViewTemplateContentType {
		t.Fatalf("view template defaults = %+v", settings.viewTemplates)
	}
	if !settings.errorHandling.enabled ||
		!settings.errorHandling.problemDetailsEnabled ||
		settings.errorHandling.path != DefaultErrorPath {
		t.Fatalf("error defaults = %+v", settings.errorHandling)
	}
	if !settings.httpClient.enabled ||
		settings.httpClient.timeout != DefaultHTTPClientTimeout ||
		settings.httpClient.maxResponseBytes != DefaultHTTPClientMaxResponseBytes ||
		len(settings.httpClient.defaultHeaders) != 0 {
		t.Fatalf("http client defaults = %+v", settings.httpClient)
	}
}

func TestNewSettings_whenEnvironmentPropertiesExist_shouldApplyWebProperties(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyApplicationName:             "admin",
		PropertyServletContextPath:          "/admin",
		PropertyServletMapping:              "/api/*",
		PropertyStaticResourcesLocations:    "resource/static, resource/public",
		PropertyStaticResourcesPattern:      "/assets/*",
		PropertyStaticResourcesServletName:  "adminStatic",
		PropertyStaticResourcesWelcomeFiles: "index.html, home.html",
		PropertyStaticResourcesEnabled:      "true",
		PropertyStaticResourcesCacheControl: "public, max-age=60",
		PropertyViewTemplatesEnabled:        "true",
		PropertyViewTemplatesLocation:       "classpath:/templates/",
		PropertyViewTemplatePrefix:          "pages",
		PropertyViewTemplateSuffix:          ".tmpl",
		PropertyViewTemplateContentType:     "text/plain; charset=utf-8",
		PropertyCORSEnabled:                 "true",
		PropertyCORSAllowedOrigins:          "https://admin.example.com",
		PropertyCORSAllowedMethods:          "GET,POST",
		PropertyCORSAllowedHeaders:          "X-Request-ID,Content-Type",
		PropertyCORSExposedHeaders:          "X-Trace-ID",
		PropertyCORSAllowCredentials:        "true",
		PropertyCORSMaxAge:                  "10m",
		PropertyForwardedHeadersEnabled:     "true",
		PropertyShallowETagEnabled:          "true",
		PropertyShallowETagMaxBodyBytes:     "4096",
		PropertyErrorEndpointEnabled:        "true",
		PropertyErrorPath:                   "/failure",
		PropertyProblemDetailsEnabled:       "true",
		PropertyHTTPClientEnabled:           "true",
		PropertyHTTPClientBaseURL:           "https://api.example.com/v1",
		PropertyHTTPClientTimeout:           "2s",
		PropertyHTTPClientMaxResponseBytes:  "8192",
		PropertyHTTPClientDefaultHeaders:    "X-App=goark, X-Trace=enabled",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if settings.applicationName != "admin" {
		t.Fatalf("application name = %q", settings.applicationName)
	}
	if settings.contextPath != "/admin" {
		t.Fatalf("context path = %q", settings.contextPath)
	}
	if settings.mappingPattern != "/api/*" {
		t.Fatalf("mapping pattern = %q", settings.mappingPattern)
	}
	if !settings.staticResources.enabled ||
		len(settings.staticResources.locations) != 2 ||
		settings.staticResources.locations[0] != "resource/static" ||
		settings.staticResources.locations[1] != "resource/public" ||
		settings.staticResources.pattern != "/assets/*" ||
		settings.staticResources.servletName != "adminStatic" ||
		len(settings.staticResources.welcomeFiles) != 2 ||
		settings.staticResources.welcomeFiles[1] != "home.html" ||
		settings.staticResources.cacheControl != "public, max-age=60" {
		t.Fatalf("static resources settings = %+v", settings.staticResources)
	}
	if !settings.filters.cors.enabled ||
		settings.filters.cors.config.AllowedOrigins[0] != "https://admin.example.com" ||
		settings.filters.cors.config.AllowedMethods[1] != "POST" ||
		settings.filters.cors.config.AllowedHeaders[0] != "X-Request-ID" ||
		settings.filters.cors.config.ExposedHeaders[0] != "X-Trace-ID" ||
		!settings.filters.cors.config.AllowCredentials ||
		settings.filters.cors.config.MaxAge != 10*time.Minute {
		t.Fatalf("cors settings = %+v", settings.filters.cors)
	}
	if !settings.filters.forwardedHeaders.enabled {
		t.Fatalf("forwarded headers settings = %+v", settings.filters.forwardedHeaders)
	}
	if !settings.viewTemplates.enabled ||
		settings.viewTemplates.location != "resource\\templates" && settings.viewTemplates.location != "resource/templates" ||
		settings.viewTemplates.prefix != "pages" ||
		settings.viewTemplates.suffix != ".tmpl" ||
		settings.viewTemplates.contentType != "text/plain; charset=utf-8" {
		t.Fatalf("view template settings = %+v", settings.viewTemplates)
	}
	if !settings.filters.shallowETag.enabled || settings.filters.shallowETag.maxBodyBytes != 4096 {
		t.Fatalf("shallow etag settings = %+v", settings.filters.shallowETag)
	}
	if !settings.errorHandling.enabled ||
		!settings.errorHandling.problemDetailsEnabled ||
		settings.errorHandling.path != "/failure" {
		t.Fatalf("error settings = %+v", settings.errorHandling)
	}
	if !settings.httpClient.enabled ||
		settings.httpClient.baseURL != "https://api.example.com/v1" ||
		settings.httpClient.timeout != 2*time.Second ||
		settings.httpClient.maxResponseBytes != 8192 ||
		settings.httpClient.defaultHeaders.Get("X-App") != "goark" ||
		settings.httpClient.defaultHeaders.Get("X-Trace") != "enabled" {
		t.Fatalf("http client settings = %+v", settings.httpClient)
	}
}

func TestNewSettings_whenOptionsExist_shouldOverrideEnvironment(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyApplicationName:          "from-env",
		PropertyServletContextPath:       "/env",
		PropertyServletMapping:           "/env/*",
		PropertyStaticResourcesLocations: "resource/env",
		PropertyStaticResourcesPattern:   "/env-static/*",
	})

	settings, err := newSettings(environment, []Option{
		WithApplicationName("from-option"),
		WithContextPath("/option"),
		WithMappingPattern("/option/*"),
		WithStaticResourceLocations("resource/option"),
		WithStaticResourcePattern("/option-static/*"),
		WithStaticResourceCacheMaxAge(2 * time.Minute),
		WithViewTemplatesLocation("resource/views"),
		WithViewTemplatePrefix("admin"),
		WithViewTemplateSuffix(".tmpl"),
		WithViewTemplateContentType("text/plain"),
		WithCORS(gowebcors.Config{AllowedOrigins: []string{"https://option.example.com"}}),
		WithForwardedHeadersEnabled(true),
		WithShallowETagEnabled(true),
		WithShallowETagMaxBodyBytes(512),
		WithErrorEndpointEnabled(true),
		WithErrorPath("/option-error"),
		WithProblemDetailsEnabled(true),
		WithHTTPClientEnabled(true),
		WithHTTPClientBaseURL("https://option.example.com/api"),
		WithHTTPClientTimeout(5 * time.Second),
		WithHTTPClientMaxResponseBytes(-1),
		WithHTTPClientDefaultHeader("X-Option", "true"),
		WithHTTPClientOptions(webclient.WithDefaultHeader("X-Custom", "yes")),
		WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
	})
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if settings.applicationName != "from-option" ||
		settings.contextPath != "/option" ||
		settings.mappingPattern != "/option/*" {
		t.Fatalf("options did not override environment: %+v", settings)
	}
	if len(settings.staticResources.locations) != 1 ||
		settings.staticResources.locations[0] != "resource/option" ||
		settings.staticResources.pattern != "/option-static/*" ||
		settings.staticResources.cacheControl != "public, max-age=120" {
		t.Fatalf("static options did not override environment: %+v", settings.staticResources)
	}
	if len(settings.arkhosOptions) != 1 {
		t.Fatalf("arkhos options count = %d, want 1", len(settings.arkhosOptions))
	}
	if settings.viewTemplates.location != "resource/views" ||
		settings.viewTemplates.prefix != "admin" ||
		settings.viewTemplates.suffix != ".tmpl" ||
		settings.viewTemplates.contentType != "text/plain" {
		t.Fatalf("view options did not override environment: %+v", settings.viewTemplates)
	}
	if !settings.filters.cors.enabled ||
		settings.filters.cors.config.AllowedOrigins[0] != "https://option.example.com" ||
		!settings.filters.forwardedHeaders.enabled ||
		!settings.filters.shallowETag.enabled ||
		settings.filters.shallowETag.maxBodyBytes != 512 {
		t.Fatalf("filter options did not override environment: %+v", settings.filters)
	}
	if !settings.errorHandling.enabled ||
		!settings.errorHandling.problemDetailsEnabled ||
		settings.errorHandling.path != "/option-error" {
		t.Fatalf("error options did not override environment: %+v", settings.errorHandling)
	}
	if !settings.httpClient.enabled ||
		settings.httpClient.baseURL != "https://option.example.com/api" ||
		settings.httpClient.timeout != 5*time.Second ||
		settings.httpClient.maxResponseBytes != -1 ||
		settings.httpClient.defaultHeaders.Get("X-Option") != "true" ||
		len(settings.httpClient.options) != 1 {
		t.Fatalf("http client options did not override environment: %+v", settings.httpClient)
	}
}

func TestNewSettings_whenSpringStaticPropertiesExist_shouldApplyCompatibleProperties(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		propertySpringStaticLocations:    "classpath:/static/,file:public",
		propertySpringStaticPattern:      "/content/*",
		propertySpringStaticCacheControl: "private, max-age=30",
		propertySpringStaticCacheMaxAge:  "10m",
		propertySpringStaticCachePeriod:  "5m",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if len(settings.staticResources.locations) != 2 ||
		settings.staticResources.locations[0] != "resource\\static" && settings.staticResources.locations[0] != "resource/static" ||
		settings.staticResources.locations[1] != "public" ||
		settings.staticResources.pattern != "/content/*" ||
		settings.staticResources.cacheControl != "private, max-age=30" {
		t.Fatalf("spring static settings = %+v", settings.staticResources)
	}
}

func TestNewSettings_whenContextPathIsInvalid_shouldReturnError(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyServletContextPath: "admin",
	})

	_, err := newSettings(environment, nil)
	if err == nil {
		t.Fatal("expected invalid context path error")
	}
}

func TestNewSettings_whenHTTPClientPropertiesAreInvalid_shouldReturnError(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
	}{
		{
			name: "base url",
			values: map[string]any{
				PropertyHTTPClientBaseURL: "://bad",
			},
		},
		{
			name: "max response bytes",
			values: map[string]any{
				PropertyHTTPClientMaxResponseBytes: "-2",
			},
		},
		{
			name: "default headers",
			values: map[string]any{
				PropertyHTTPClientDefaultHeaders: "X-App",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSettings(newTestEnvironment(t, tt.values), nil)
			if err == nil {
				t.Fatal("expected invalid http client property error")
			}
		})
	}
}

func TestRegisterHTTPClient_whenDisabledOrExistingBeans_shouldBackOff(t *testing.T) {
	registry := goarkcontainer.NewRegistry()
	if err := registerHTTPClient(registry, httpClientSettings{enabled: false}); err != nil {
		t.Fatalf("register disabled client failed: %v", err)
	}
	if _, exists := registry.Definition(BeanNameHTTPClient); exists {
		t.Fatal("disabled http client should not register client bean")
	}
	if _, exists := registry.Definition(BeanNameHTTPClientBuilder); exists {
		t.Fatal("disabled http client should not register builder bean")
	}

	customBuilder := webclient.NewBuilder(webclient.WithDefaultHeader("X-Custom", "builder"))
	if err := goarkcontainer.RegisterInstance[*webclient.Builder](registry, BeanNameHTTPClientBuilder, customBuilder); err != nil {
		t.Fatalf("register custom builder failed: %v", err)
	}
	customClient, err := webclient.New(webclient.WithDefaultHeader("X-Custom", "client"))
	if err != nil {
		t.Fatalf("new custom client failed: %v", err)
	}
	if err := goarkcontainer.RegisterInstance[*webclient.Client](registry, BeanNameHTTPClient, customClient); err != nil {
		t.Fatalf("register custom client failed: %v", err)
	}
	if err := registerHTTPClient(registry, defaultHTTPClientSettings()); err != nil {
		t.Fatalf("register default client failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("new runtime container failed: %v", err)
	}
	resolvedBuilder, err := goarkcontainer.Get[*webclient.Builder](context.Background(), runtimeContainer, BeanNameHTTPClientBuilder)
	if err != nil {
		t.Fatalf("resolve custom builder failed: %v", err)
	}
	if resolvedBuilder != customBuilder {
		t.Fatal("default registration replaced custom builder")
	}
	resolvedClient, err := goarkcontainer.Get[*webclient.Client](context.Background(), runtimeContainer, BeanNameHTTPClient)
	if err != nil {
		t.Fatalf("resolve custom client failed: %v", err)
	}
	if resolvedClient != customClient {
		t.Fatal("default registration replaced custom client")
	}
}

func newTestEnvironment(t *testing.T, values map[string]any) coreenv.Environment {
	t.Helper()

	environment, err := coreenv.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("new environment failed: %v", err)
	}
	source, err := coreenv.NewMapPropertySource("test", values)
	if err != nil {
		t.Fatalf("new property source failed: %v", err)
	}
	if err := environment.PropertySources().AddFirst(source); err != nil {
		t.Fatalf("add property source failed: %v", err)
	}
	return environment
}
