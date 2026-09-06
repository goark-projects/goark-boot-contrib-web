package gbcweb_test

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenWebFiltersConfigured_shouldApplyCorsForwardedHeadersAndETag(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
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
			mvc.GET("/filters", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterForwardedPayload, error) {
				ctx.Response().Header().Set("X-Trace-ID", "trace-1")
				return starterForwardedPayload{
					URL:    ctx.Request().RequestURL(),
					Remote: ctx.Request().RemoteAddr(),
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

func TestAutoConfigure_whenHiddenHTTPMethodFilterEnabled_shouldRouteOverriddenMethod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  mvc:
    hiddenmethod:
      filter:
        enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.hidden-method", mvc.NewRestController("items",
			mvc.POST("/items/1", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
				return http.MethodPost, nil
			})),
			mvc.DELETE("/items/1", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
				return http.MethodDelete, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/items/1", strings.NewReader("_method=DELETE"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return request, nil
	}, http.StatusOK)
	if snapshot.body != http.MethodDelete {
		t.Fatalf("body = %q, want DELETE route", snapshot.body)
	}
}

func TestAutoConfigure_whenFormContentFilterEnabled_shouldBindDeleteFormParameters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  mvc:
    formcontent:
      filter:
        enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.form-content", mvc.NewRestController("items",
			mvc.DELETE("/items/1", mvc.Return(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
				return mvc.RequestParamString(ctx, "name")
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, starterServerURL(t, app)+"/items/1", strings.NewReader("name=goark"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return request, nil
	}, http.StatusOK)
	if snapshot.body != "goark" {
		t.Fatalf("body = %q, want form parameter", snapshot.body)
	}
}

func TestAutoConfigure_whenCharacterEncodingFilterEnabled_shouldApplyEncoding(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  servlet:
    encoding:
      charset: UTF-8
      force-response: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.encoding", mvc.NewRestController("encoding",
			mvc.POST("/encoding/request", mvc.Return(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
				return ctx.Request().CharacterEncoding(), nil
			})),
			mvc.GET("/encoding/response", mvc.Entity(func(_ *arkweb.Context) (goweb.ResponseEntity[string], error) {
				return goweb.Status(http.StatusOK, "ok").WithContentType("text/plain; charset=iso-8859-1"), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	requestSnapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/encoding/request", strings.NewReader("{}"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		return request, nil
	}, http.StatusOK)
	if !strings.EqualFold(requestSnapshot.body, "UTF-8") {
		t.Fatalf("request encoding body = %q, want UTF-8", requestSnapshot.body)
	}
	responseSnapshot := requestUntilStatusSnapshot(t, serverURL+"/encoding/response", http.StatusOK)
	if got := responseSnapshot.header.Get("Content-Type"); !strings.Contains(strings.ToLower(got), "charset=utf-8") {
		t.Fatalf("response Content-Type = %q, want UTF-8 charset", got)
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
	return requestUntilStatusSnapshotWithMethod(t, http.MethodGet, target, statusCode)
}

func requestUntilStatusSnapshotWithMethod(t *testing.T, method string, target string, statusCode int) responseSnapshot {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(3 * time.Second)
	for {
		request, requestErr := http.NewRequestWithContext(t.Context(), method, target, nil)
		if requestErr != nil {
			t.Fatalf("%s %s request invalid: %v", method, target, requestErr)
		}
		response, err := client.Do(request)
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
			t.Fatalf("%s %s did not succeed before deadline: %v", method, target, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type starterErrorMapperConfiguration struct{}

type starterErrorMapperDependency struct{}

type starterDependentErrorMapperConfiguration struct{}

func (starterDependentErrorMapperConfiguration) Name() string {
	return "test.web.dependent-error-mapper"
}

func (starterDependentErrorMapperConfiguration) Order() int {
	return 0
}

func (c starterDependentErrorMapperConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}
