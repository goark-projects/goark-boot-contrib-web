package gbcweb_test

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/web/mvc"
	mvcview "goark.dev/goark/web/mvc/view"
)

func TestAutoConfigure_whenWebFiltersConfigured_shouldApplyCorsForwardedHeadersAndETag(
	t *testing.T,
) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.filters", mvc.NewController(
			"filters",
			mvc.GET(
				"/filters",
				mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterForwardedPayload, error) {
					ctx.Response().Header().Set("X-Trace-ID", "trace-1")
					return starterForwardedPayload{
						URL:    ctx.Request().RequestURL(),
						Remote: ctx.Request().RemoteAddr(),
					}, nil
				}),
			),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	preflight := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodOptions,
			serverURL+"/filters",
			nil,
		)
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
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/filters",
			nil,
		)
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
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/filters",
			nil,
		)
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

func TestAutoConfigure_whenControllerAndRestControllerReturnValuesExist_shouldUseDefaultStrategies(
	t *testing.T,
) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates")
	mkdir(t, templateDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(templateDir, "home.html"), "<h1>home</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.controller-kind",
			mvc.NewController("pages",
				mvc.GET("/home", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
					return "home", nil
				})),
			),
			mvc.NewRestController(
				"api",
				mvc.GET(
					"/api/status",
					mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
						return "UP", nil
					}),
				),
			),
		)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	viewSnapshot := requestUntilStatusSnapshot(t, serverURL+"/home", http.StatusOK)
	if viewSnapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("view Content-Type = %q, want html", viewSnapshot.header.Get("Content-Type"))
	}
	if viewSnapshot.body != "<h1>home</h1>" {
		t.Fatalf("view body = %q, want rendered view", viewSnapshot.body)
	}
	restSnapshot := requestUntilStatusSnapshot(t, serverURL+"/api/status", http.StatusOK)
	if restSnapshot.header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("rest Content-Type = %q, want text/plain", restSnapshot.header.Get("Content-Type"))
	}
	if restSnapshot.body != "UP" {
		t.Fatalf("rest body = %q, want raw response body", restSnapshot.body)
	}
}

func TestAutoConfigure_whenModelViewRedirectHasAttributes_shouldExpandLocation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.redirect-attributes", mvc.NewController(
				"redirects",
				mvc.GET(
					"/accounts",
					mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
						model := mvc.NewModel().
							AddAttribute("id", "a/b").
							AddAttribute("page", 2).
							AddAttribute("tab", "security")
						return mvc.NewModelAndView(
							"redirect:/users/{id}",
							model,
							mvc.WithViewStatus(http.StatusSeeOther),
						), nil
					}),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWithClient(t, http.Client{
		Timeout: time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/accounts", nil)
	}, http.StatusSeeOther)
	if got := snapshot.header.Get("Location"); got != "/users/a%2Fb?page=2&tab=security" {
		t.Fatalf("Location = %q, want expanded redirect", got)
	}
}
func TestAutoConfigure_whenDefaultTemplateExists_shouldRenderMVCView(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates")
	mkdir(t, templateDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(templateDir, "home.html"), "<h1>{{.Title}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.view", mvc.NewController("views",
			mvc.GET("/home", mvc.Handler(func(_ *arkweb.Context) (arkweb.Result, error) {
				return mvcview.Render("home", map[string]string{"Title": "Goark"}), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)
	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/home", http.StatusOK)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>Goark</h1>" {
		t.Fatalf("view body = %q, want rendered html", snapshot.body)
	}
}

func starterFileOnlyMultipartBody(t testing.TB) (string, string) {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "profile.txt")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := io.WriteString(part, "hello"); err != nil {
		t.Fatalf("write part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	return body.String(), writer.FormDataContentType()
}

type starterMatrixVariableValuesPayload struct {
	Colors      []string            `json:"colors"`
	Codes       []int               `json:"codes"`
	OwnerColors []string            `json:"ownerColors"`
	Matrix      map[string]string   `json:"matrix"`
	Values      map[string][]string `json:"values"`
	OwnerMatrix map[string]string   `json:"ownerMatrix"`
	OwnerValues map[string][]string `json:"ownerValues"`
}

func (c starterResponseAdviceConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterRequestParamPayload struct {
	Query string   `json:"query"`
	Tags  []string `json:"tags"`
	IDs   []int64  `json:"ids"`
}

type starterOptionalBodyPayload struct {
	Present bool   `json:"present"`
	Name    string `json:"name"`
}

type starterArrayPayload struct {
	Tags    []string `json:"tags"`
	Aliases []string `json:"aliases"`
}

func (starterWebSocketConfiguration) Name() string {
	return "test.web.websocket"
}

type starterLocaleChangeConfiguration struct{}
