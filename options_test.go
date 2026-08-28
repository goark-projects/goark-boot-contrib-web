package gbcweb

import (
	"testing"

	gbcarkhos "goark.dev/gbc-arkhos"
	coreenv "goark.dev/goark/core/env"
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
	if !settings.staticResources.enabled ||
		len(settings.staticResources.locations) != 1 ||
		settings.staticResources.locations[0] != DefaultStaticResourcesLocation ||
		settings.staticResources.pattern != DefaultStaticResourcesPattern {
		t.Fatalf("static resources defaults = %+v", settings.staticResources)
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
		settings.staticResources.welcomeFiles[1] != "home.html" {
		t.Fatalf("static resources settings = %+v", settings.staticResources)
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
		settings.staticResources.pattern != "/option-static/*" {
		t.Fatalf("static options did not override environment: %+v", settings.staticResources)
	}
	if len(settings.arkhosOptions) != 1 {
		t.Fatalf("arkhos options count = %d, want 1", len(settings.arkhosOptions))
	}
}

func TestNewSettings_whenSpringStaticPropertiesExist_shouldApplyCompatibleProperties(t *testing.T) {
	environment := newTestEnvironment(t, map[string]any{
		propertySpringStaticLocations: "classpath:/static/,file:public",
		propertySpringStaticPattern:   "/content/*",
	})

	settings, err := newSettings(environment, nil)
	if err != nil {
		t.Fatalf("new settings failed: %v", err)
	}

	if len(settings.staticResources.locations) != 2 ||
		settings.staticResources.locations[0] != "resource\\static" && settings.staticResources.locations[0] != "resource/static" ||
		settings.staticResources.locations[1] != "public" ||
		settings.staticResources.pattern != "/content/*" {
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
