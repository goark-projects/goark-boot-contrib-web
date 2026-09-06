package gbcweb

import (
	"testing"
	"time"
)

func TestModuleMetadata(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "module path", got: ModulePath, want: "goark.dev/gbc-web"},
		{name: "repository", got: Repository, want: "goark-boot-contrib-web"},
		{name: "starter id", got: StarterID, want: "goark.boot.contrib.web"},
		{name: "deployment bean", got: BeanNameDeployment, want: "goark.boot.web.deployment"},
		{name: "message reader bean", got: BeanNameMessageReader, want: "goark.boot.web.messageReader"},
		{name: "message writer bean", got: BeanNameMessageWriter, want: "goark.boot.web.messageWriter"},
		{name: "message io configurer bean", got: BeanNameMessageIOConfigurer, want: "goark.boot.web.messageIOConfigurer"},
		{name: "validator bean", got: BeanNameValidator, want: "goark.boot.web.validator"},
		{name: "validator configurer bean", got: BeanNameValidatorConfigurer, want: "goark.boot.web.validatorConfigurer"},
		{name: "conversion service bean", got: BeanNameConversionService, want: "goark.boot.web.mvc.conversionService"},
		{name: "conversion configurer bean", got: BeanNameConversionConfigurer, want: "goark.boot.web.mvc.conversionConfigurer"},
		{name: "http client builder bean", got: BeanNameHTTPClientBuilder, want: "goark.boot.web.clientBuilder"},
		{name: "http client bean", got: BeanNameHTTPClient, want: "goark.boot.web.client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("metadata mismatch: got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

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
		settings.staticResources.pattern != DefaultStaticResourcesPattern ||
		settings.staticResources.contentVersion != DefaultStaticResourceContentVersioningEnabled ||
		settings.staticResources.fixedVersion != DefaultStaticResourceFixedVersion {
		t.Fatalf("static resources defaults = %+v", settings.staticResources)
	}
	if settings.filters.cors.enabled ||
		settings.filters.forwardedHeaders.enabled ||
		settings.filters.hiddenMethod.enabled ||
		settings.filters.shallowETag.enabled {
		t.Fatalf("filter defaults = %+v", settings.filters)
	}
	if !settings.filters.flashMap.enabled || settings.filters.flashMap.timeout != DefaultFlashMapTimeout {
		t.Fatalf("flash map defaults = %+v", settings.filters.flashMap)
	}
	if !settings.filters.sessionAttributes.enabled {
		t.Fatalf("session attributes defaults = %+v", settings.filters.sessionAttributes)
	}
	if !settings.filters.characterEncoding.enabled ||
		settings.filters.characterEncoding.encoding != DefaultCharacterEncoding ||
		!settings.filters.characterEncoding.forceRequest ||
		settings.filters.characterEncoding.forceResponse {
		t.Fatalf("character encoding defaults = %+v", settings.filters.characterEncoding)
	}
	if !settings.filters.formContent.enabled || settings.filters.formContent.maxBodyBytes != DefaultFormContentMaxBodyBytes {
		t.Fatalf("form content defaults = %+v", settings.filters.formContent)
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

func TestNewSettings_whenApplicationNameConfigured_shouldUseDirectGoarkNamespace(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		"goark.application.name":     "admin",
		"goark.web.application.name": "legacy",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}
	if PropertyApplicationName != "goark.application.name" {
		t.Fatalf("application name property = %q", PropertyApplicationName)
	}
	if settings.applicationName != "admin" {
		t.Fatalf("application name = %q, want admin", settings.applicationName)
	}
}

func TestNewSettings_whenOnlyLegacyWebApplicationNameExists_shouldIgnoreIt(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		"goark.web.application.name": "legacy",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}
	if settings.applicationName != DefaultApplicationName {
		t.Fatalf("application name = %q, want default %q", settings.applicationName, DefaultApplicationName)
	}
}

func TestNewSettings_whenEnvironmentPropertiesExist_shouldApplyWebProperties(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		PropertyApplicationName:                        "admin",
		PropertyServletContextPath:                     "/admin",
		PropertyServletMapping:                         "/api/*",
		PropertyStaticResourcesLocations:               "resource/static, resource/public",
		PropertyStaticResourcesPattern:                 "/assets/*",
		PropertyStaticResourcesServletName:             "adminStatic",
		PropertyStaticResourcesWelcomeFiles:            "index.html, home.html",
		PropertyStaticResourcesEnabled:                 "true",
		PropertyStaticResourcesCacheControl:            "public, max-age=60",
		PropertyStaticResourceContentVersioningEnabled: "true",
		PropertyStaticResourceFixedVersion:             "v1",
		PropertyViewTemplatesEnabled:                   "true",
		PropertyViewTemplatesLocation:                  "classpath:/templates/",
		PropertyViewTemplatePrefix:                     "pages",
		PropertyViewTemplateSuffix:                     ".tmpl",
		PropertyViewTemplateContentType:                "text/plain; charset=utf-8",
		PropertyCORSEnabled:                            "true",
		PropertyCORSAllowedOrigins:                     "https://admin.example.com",
		PropertyCORSAllowedMethods:                     "GET,POST",
		PropertyCORSAllowedHeaders:                     "X-Request-ID,Content-Type",
		PropertyCORSExposedHeaders:                     "X-Trace-ID",
		PropertyCORSAllowCredentials:                   "true",
		PropertyCORSMaxAge:                             "10m",
		PropertyForwardedHeadersEnabled:                "true",
		PropertyCharacterEncodingFilterEnabled:         "true",
		PropertyCharacterEncoding:                      "GBK",
		PropertyForceCharacterEncoding:                 "false",
		PropertyForceRequestCharacterEncoding:          "true",
		PropertyForceResponseCharacterEncoding:         "true",
		PropertyShallowETagEnabled:                     "true",
		PropertyShallowETagMaxBodyBytes:                "4096",
		PropertyHiddenHTTPMethodFilterEnabled:          "true",
		PropertyFormContentFilterEnabled:               "true",
		PropertyFormContentMaxBodyBytes:                "2048",
		PropertyFlashMapFilterEnabled:                  "true",
		PropertyFlashMapTimeout:                        "45s",
		PropertySessionAttributesFilterEnabled:         "false",
		PropertyErrorEndpointEnabled:                   "true",
		PropertyErrorPath:                              "/failure",
		PropertyProblemDetailsEnabled:                  "true",
		PropertyHTTPClientEnabled:                      "true",
		PropertyHTTPClientBaseURL:                      "https://api.example.com/v1",
		PropertyHTTPClientTimeout:                      "2s",
		PropertyHTTPClientMaxResponseBytes:             "8192",
		PropertyHTTPClientDefaultHeaders:               "X-App=goark, X-Trace=enabled",
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
		settings.staticResources.cacheControl != "public, max-age=60" ||
		!settings.staticResources.contentVersion ||
		settings.staticResources.fixedVersion != "v1" {
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
	if !settings.filters.characterEncoding.enabled ||
		settings.filters.characterEncoding.encoding != "GBK" ||
		!settings.filters.characterEncoding.forceRequest ||
		!settings.filters.characterEncoding.forceResponse {
		t.Fatalf("character encoding settings = %+v", settings.filters.characterEncoding)
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
	if !settings.filters.hiddenMethod.enabled {
		t.Fatalf("hidden method settings = %+v", settings.filters.hiddenMethod)
	}
	if !settings.filters.formContent.enabled || settings.filters.formContent.maxBodyBytes != 2048 {
		t.Fatalf("form content settings = %+v", settings.filters.formContent)
	}
	if !settings.filters.flashMap.enabled || settings.filters.flashMap.timeout != 45*time.Second {
		t.Fatalf("flash map settings = %+v", settings.filters.flashMap)
	}
	if settings.filters.sessionAttributes.enabled {
		t.Fatalf("session attributes settings = %+v", settings.filters.sessionAttributes)
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
