package gbcweb

import (
	"context"
	"net/http"
	"testing"
	"time"

	gbcarkhos "goark.dev/gbc-arkhos"
	goarkcontainer "goark.dev/goark/container"
	webclient "goark.dev/goark/web/client"
	gowebcors "goark.dev/goark/web/cors"
	gowebfilter "goark.dev/goark/web/filter"
)

func TestNewSettings_whenOptionsExist_shouldOverrideEnvironment(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyApplicationName:                "from-env",
		PropertyServletContextPath:             "/env",
		PropertyServletMapping:                 "/env/*",
		PropertyStaticResourcesLocations:       "resource/env",
		PropertyStaticResourcesPattern:         "/env-static/*",
		PropertyFormContentFilterEnabled:       "false",
		PropertySessionAttributesFilterEnabled: "false",
		PropertyCharacterEncoding:              "GBK",
	})

	settings, err := newSettings(environment, []Option{
		WithApplicationName("from-option"),
		WithContextPath("/option"),
		WithMappingPattern("/option/*"),
		WithStaticResourceLocations("resource/option"),
		WithStaticResourcePattern("/option-static/*"),
		WithStaticResourceCacheMaxAge(2 * time.Minute),
		WithStaticResourceContentVersioning(true),
		WithStaticResourceFixedVersion("release-1"),
		WithViewTemplatesLocation("resource/views"),
		WithViewTemplatePrefix("admin"),
		WithViewTemplateSuffix(".tmpl"),
		WithViewTemplateContentType("text/plain"),
		WithCORS(gowebcors.Config{AllowedOrigins: []string{"https://option.example.com"}}),
		WithForwardedHeadersEnabled(true),
		WithCharacterEncoding("UTF-16"),
		WithForceCharacterEncoding(false),
		WithForceRequestCharacterEncoding(true),
		WithForceResponseCharacterEncoding(true),
		WithCharacterEncodingFilterOptions(gowebfilter.WithCharacterEncoding("UTF-8")),
		WithShallowETagEnabled(true),
		WithShallowETagMaxBodyBytes(512),
		WithHiddenHTTPMethodFilterEnabled(true),
		WithHiddenHTTPMethodFilterOptions(gowebfilter.WithHiddenMethodParameter("http_method")),
		WithFormContentFilterEnabled(true),
		WithFormContentMaxBodyBytes(1024),
		WithFormContentFilterOptions(gowebfilter.WithFormContentMethods(http.MethodDelete)),
		WithFlashMapFilterEnabled(true),
		WithFlashMapTimeout(90 * time.Second),
		WithSessionAttributesFilterEnabled(true),
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
		settings.staticResources.cacheControl != "public, max-age=120" ||
		!settings.staticResources.contentVersion ||
		settings.staticResources.fixedVersion != "release-1" {
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
		!settings.filters.characterEncoding.enabled ||
		settings.filters.characterEncoding.encoding != "UTF-16" ||
		!settings.filters.characterEncoding.forceRequest ||
		!settings.filters.characterEncoding.forceResponse ||
		len(settings.filters.characterEncoding.options) != 1 ||
		!settings.filters.hiddenMethod.enabled ||
		len(settings.filters.hiddenMethod.options) != 1 ||
		!settings.filters.formContent.enabled ||
		settings.filters.formContent.maxBodyBytes != 1024 ||
		len(settings.filters.formContent.options) != 1 ||
		!settings.filters.flashMap.enabled ||
		settings.filters.flashMap.timeout != 90*time.Second ||
		!settings.filters.sessionAttributes.enabled ||
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

func TestNewSettings_whenStaticPropertiesExist_shouldApplyProperties(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyStaticResourcesLocations:               "classpath:/static/,file:public",
		PropertyStaticResourcesPattern:                 "/content/*",
		PropertyStaticResourcesCacheControl:            "private, max-age=30",
		PropertyStaticResourcesCacheMaxAge:             "10m",
		PropertyStaticResourceContentVersioningEnabled: "true",
		PropertyStaticResourceFixedVersion:             "v2",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if len(settings.staticResources.locations) != 2 ||
		settings.staticResources.locations[0] != "resource\\static" &&
			settings.staticResources.locations[0] != "resource/static" ||
		settings.staticResources.locations[1] != "public" ||
		settings.staticResources.pattern != "/content/*" ||
		settings.staticResources.cacheControl != "private, max-age=30" ||
		!settings.staticResources.contentVersion ||
		settings.staticResources.fixedVersion != "v2" {
		t.Fatalf("static settings = %+v", settings.staticResources)
	}
}

func TestNewSettings_whenEncodingPropertiesExist_shouldApplyProperties(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyCharacterEncodingFilterEnabled: "true",
		PropertyCharacterEncoding:              "UTF-16",
		PropertyForceCharacterEncoding:         "false",
		PropertyForceRequestCharacterEncoding:  "true",
		PropertyForceResponseCharacterEncoding: "true",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if !settings.filters.characterEncoding.enabled ||
		settings.filters.characterEncoding.encoding != "UTF-16" ||
		!settings.filters.characterEncoding.forceRequest ||
		!settings.filters.characterEncoding.forceResponse {
		t.Fatalf("character encoding settings = %+v", settings.filters.characterEncoding)
	}
}

func TestNewSettings_whenFormContentPropertyExists_shouldApplyProperty(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyFormContentFilterEnabled: "false",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if settings.filters.formContent.enabled {
		t.Fatalf("form content settings = %+v", settings.filters.formContent)
	}
}

func TestNewSettings_whenHiddenMethodPropertyExists_shouldApplyProperty(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyHiddenHTTPMethodFilterEnabled: "true",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if !settings.filters.hiddenMethod.enabled {
		t.Fatalf("hidden method settings = %+v", settings.filters.hiddenMethod)
	}
}

func TestNewSettings_whenFlashMapTimeoutPropertyIsBlank_shouldKeepDefault(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyFlashMapTimeout: " ",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if settings.filters.flashMap.timeout != DefaultFlashMapTimeout {
		t.Fatalf(
			"flash map timeout = %s, want %s",
			settings.filters.flashMap.timeout,
			DefaultFlashMapTimeout,
		)
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
			name: "form content max body bytes",
			values: map[string]any{
				PropertyFormContentMaxBodyBytes: "-1",
			},
		},
		{
			name: "flash map timeout",
			values: map[string]any{
				PropertyFlashMapTimeout: "-1s",
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
	if err := goarkcontainer.RegisterInstance[*webclient.Builder](
		registry, BeanNameHTTPClientBuilder, customBuilder,
	); err != nil {
		t.Fatalf("register custom builder failed: %v", err)
	}
	customClient, err := webclient.New(webclient.WithDefaultHeader("X-Custom", "client"))
	if err != nil {
		t.Fatalf("new custom client failed: %v", err)
	}
	if err := goarkcontainer.RegisterInstance[*webclient.Client](
		registry, BeanNameHTTPClient, customClient,
	); err != nil {
		t.Fatalf("register custom client failed: %v", err)
	}
	if err := registerHTTPClient(registry, defaultHTTPClientSettings()); err != nil {
		t.Fatalf("register default client failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("new runtime container failed: %v", err)
	}
	resolvedBuilder, err := goarkcontainer.Get[*webclient.Builder](
		context.Background(),
		runtimeContainer,
		BeanNameHTTPClientBuilder,
	)
	if err != nil {
		t.Fatalf("resolve custom builder failed: %v", err)
	}
	if resolvedBuilder != customBuilder {
		t.Fatal("default registration replaced custom builder")
	}
	resolvedClient, err := goarkcontainer.Get[*webclient.Client](
		context.Background(),
		runtimeContainer,
		BeanNameHTTPClient,
	)
	if err != nil {
		t.Fatalf("resolve custom client failed: %v", err)
	}
	if resolvedClient != customClient {
		t.Fatal("default registration replaced custom client")
	}
}
