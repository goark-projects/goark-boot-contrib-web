package gbcweb_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

var errStarterMapped = errors.New("starter mapped")

func TestAutoConfigure_whenMVCControllerExists_shouldServeRequestWithArkhos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc", mvc.NewController("health",
			mvc.GET("/healthz", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
				return map[string]string{"status": "UP"}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	body := requestUntilOK(t, server.URL()+"/healthz")
	if body != `{"status":"UP"}` {
		t.Fatalf("body = %q, want health json", body)
	}
}

func TestAutoConfigure_whenDefaultResourceStaticExists_shouldServeStaticResources(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	staticDir := filepath.Join(resource, "static")
	mkdir(t, staticDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)
	writeFile(t, filepath.Join(staticDir, "app.txt"), "resource static")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	if body := requestUntilOK(t, serverURL+"/static/app.txt"); body != "resource static" {
		t.Fatalf("static body = %q, want resource static", body)
	}
}

func TestAutoConfigure_whenStaticResourcesConfigured_shouldServeConfiguredLocationAndPattern(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	publicDir := filepath.Join(root, "public")
	mkdir(t, resource)
	mkdir(t, publicDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
    resources:
      static:
        locations: public
        pattern: /assets/*
        welcome-files: index.html
`)
	writeFile(t, filepath.Join(publicDir, "app.txt"), "configured static")
	writeFile(t, filepath.Join(publicDir, "index.html"), "configured index")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	if body := requestUntilOK(t, serverURL+"/assets/app.txt"); body != "configured static" {
		t.Fatalf("configured static body = %q", body)
	}
	if body := requestUntilOK(t, serverURL+"/assets/"); body != "configured index" {
		t.Fatalf("configured welcome body = %q", body)
	}
}

func TestAutoConfigure_whenResponseEntityRouteExists_shouldPreserveStatusHeadersAndBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.entity", mvc.NewController("jobs",
			mvc.GET("/jobs/1", mvc.Entity(func(_ *arkweb.Context) (goweb.ResponseEntity[map[string]string], error) {
				return goweb.Status(http.StatusAccepted, map[string]string{"state": "queued"}).
					WithHeader("X-Starter-Entity", "true"), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	snapshot := requestUntilStatusSnapshot(t, server.URL()+"/jobs/1", http.StatusAccepted)
	if snapshot.header.Get("X-Starter-Entity") != "true" {
		t.Fatalf("X-Starter-Entity = %q, want true", snapshot.header.Get("X-Starter-Entity"))
	}
	if snapshot.body != `{"state":"queued"}` {
		t.Fatalf("body = %q, want entity json", snapshot.body)
	}
}

func TestAutoConfigure_whenWebErrorMapperConfigurerExists_shouldApplyMapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc", mvc.NewController("errors",
				mvc.GET("/errors", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return nil, errStarterMapped
				})),
			)),
			starterErrorMapperConfiguration{},
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	body := requestUntilStatus(t, server.URL()+"/errors", http.StatusConflict)
	if body != "mapped" {
		t.Fatalf("body = %q, want mapped", body)
	}
}

func TestAutoConfigure_whenWebFiltersConfigured_shouldApplyCorsForwardedHeadersAndETag(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
    cors:
      enabled: true
      allowed-origins: https://admin.example.com
      allowed-methods: GET,POST
      allowed-headers: X-Request-ID,Content-Type
      exposed-headers: X-Trace-ID
      allow-credentials: true
      max-age: 5m
    filters:
      forwarded-headers:
        enabled: true
      shallow-etag:
        enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.filters", mvc.NewController("filters",
			mvc.GET("/filters", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
				ctx.Response().Header().Set("X-Trace-ID", "trace-1")
				return map[string]string{
					"url":    ctx.Request().RequestURL(),
					"remote": ctx.Request().RemoteAddr(),
				}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	preflight := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, serverURL+"/filters", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		request.Header.Set("Access-Control-Request-Headers", "x-request-id, content-type")
		return request, nil
	}, http.StatusNoContent)
	if preflight.header.Get("Access-Control-Allow-Origin") != "https://admin.example.com" ||
		preflight.header.Get("Access-Control-Allow-Credentials") != "true" ||
		preflight.header.Get("Access-Control-Max-Age") != "300" {
		t.Fatalf("preflight headers = %#v", preflight.header)
	}

	first := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/filters", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "api.example.com")
		request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
		return request, nil
	}, http.StatusOK)
	if first.header.Get("Access-Control-Allow-Origin") != "https://admin.example.com" ||
		first.header.Get("Access-Control-Expose-Headers") != "X-Trace-ID" ||
		first.header.Get("ETag") == "" {
		t.Fatalf("actual headers = %#v", first.header)
	}
	if !strings.Contains(first.body, `"url":"https://api.example.com/filters"`) ||
		!strings.Contains(first.body, `"remote":"203.0.113.7"`) {
		t.Fatalf("forwarded body = %q", first.body)
	}

	second := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/filters", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "api.example.com")
		request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
		request.Header.Set("If-None-Match", first.header.Get("ETag"))
		return request, nil
	}, http.StatusNotModified)
	if second.body != "" {
		t.Fatalf("not modified body = %q, want empty", second.body)
	}
}

func TestAutoConfigure_whenRegisteredTwice_shouldBackOffExistingConfigurations(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(
			gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0"))),
			gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0"))),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	assertConfigurationCount(t, appContext.Configurations(), "goark.boot.contrib.web.configuration", 1)
	assertConfigurationCount(t, appContext.Configurations(), "goark.boot.contrib.arkhos.configuration", 1)
}

func requestUntilOK(t *testing.T, target string) string {
	return requestUntilStatus(t, target, http.StatusOK)
}

func requestUntilStatus(t *testing.T, target string, statusCode int) string {
	return requestUntilStatusSnapshot(t, target, statusCode).body
}

type responseSnapshot struct {
	body   string
	header http.Header
}

func requestUntilStatusSnapshot(t *testing.T, target string, statusCode int) responseSnapshot {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, err := client.Get(target)
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read/close response = %v/%v", readErr, closeErr)
			}
			if response.StatusCode == statusCode {
				return responseSnapshot{
					body:   string(body),
					header: response.Header.Clone(),
				}
			}
			t.Fatalf("status = %d, want %d, body = %q", response.StatusCode, statusCode, string(body))
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET %s did not succeed before deadline: %v", target, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type starterErrorMapperConfiguration struct{}

func (starterErrorMapperConfiguration) Name() string {
	return "test.web.error-mapper"
}

func (starterErrorMapperConfiguration) Order() int {
	return 0
}

func (c starterErrorMapperConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterErrorMapperConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return goweb.RegisterErrorMapper(config.Registry(), "testErrorMapper", goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
		if errors.Is(err, errStarterMapped) {
			return arkweb.Text(http.StatusConflict, "mapped")
		}
		return nil
	}))
}

func closeApp(t *testing.T, app *boot.Application) {
	t.Helper()
	if err := app.Close(t.Context()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q failed: %v", path, err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q failed: %v", path, err)
	}
}

func clearConfigDataEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		configdata.EnvConfigLocation,
		configdata.EnvConfigName,
		configdata.EnvProfilesActive,
		"SPRING_CONFIG_LOCATION",
		"SPRING_CONFIG_NAME",
		"SPRING_PROFILES_ACTIVE",
	} {
		t.Setenv(name, "")
	}
}

func assertConfigurationCount(t *testing.T, descriptors []goark.ConfigurationDescriptor, name string, want int) {
	t.Helper()

	got := 0
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			got++
		}
	}
	if got != want {
		t.Fatalf("configuration %q count = %d, want %d", name, got, want)
	}
}
