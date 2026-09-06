package gbcweb_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	gowebstatic "goark.dev/goark/web/static"
)

func TestAutoConfigure_whenStaticResourcesConfigured_shouldServeConfiguredLocationAndPattern(
	t *testing.T,
) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	publicDir := filepath.Join(root, "public")
	mkdir(t, resource)
	mkdir(t, publicDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
    resources:
      static-locations: public
      static:
        welcome-files: index.html
      chain:
        strategy:
          content:
            enabled: true
          fixed:
            version: v1
      cache:
        cachecontrol:
          max-age: 1h
  mvc:
    static-path-pattern: /assets/*
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
	staticSnapshot := requestUntilStatusSnapshot(t, serverURL+"/assets/app.txt", http.StatusOK)
	if staticSnapshot.body != "configured static" {
		t.Fatalf("configured static body = %q", staticSnapshot.body)
	}
	if got := staticSnapshot.header.Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("Cache-Control = %q, want public max-age", got)
	}
	if body := requestUntilOK(t, serverURL+"/assets/"); body != "configured index" {
		t.Fatalf("configured welcome body = %q", body)
	}
	versioned, err := gowebstatic.ContentVersionPath(t.Context(), os.DirFS(publicDir), "app.txt")
	if err != nil {
		t.Fatalf("ContentVersionPath failed: %v", err)
	}
	versioned, err = gowebstatic.FixedVersionPath("v1", versioned)
	if err != nil {
		t.Fatalf("FixedVersionPath failed: %v", err)
	}
	if body := requestUntilOK(t, serverURL+"/assets/"+versioned); body != "configured static" {
		t.Fatalf("versioned static body = %q", body)
	}
	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	provider, err := goark.Get[gowebstatic.ResourceURLProvider](
		t.Context(),
		appContext,
		gbcweb.BeanNameStaticResourceURLProvider,
	)
	if err != nil {
		t.Fatalf("resolve static resource url provider failed: %v", err)
	}
	resourceURL, err := provider.URL(t.Context(), "app.txt")
	if err != nil {
		t.Fatalf("provider URL failed: %v", err)
	}
	if !strings.HasPrefix(resourceURL, "/assets/v1/app-") ||
		!strings.HasSuffix(resourceURL, ".txt") {
		t.Fatalf("resource url = %q, want versioned assets URL", resourceURL)
	}
	if body := requestUntilOK(t, serverURL+resourceURL); body != "configured static" {
		t.Fatalf("provider static body = %q", body)
	}
}

func TestAutoConfigure_whenControllerAdviceMessageAdviceExists_shouldServeThroughArkhos(
	t *testing.T,
) {
	advice := mvc.NewRestControllerAdvice("test.mvc.message-advice").WithRequestBodyAdvice(
		goweb.RequestBodyAdviceFunc{
			After: func(_ *arkweb.Context, input goweb.RequestBodyAdviceContext) error {
				target := input.Target.(*starterAdviceBodyInput)
				target.Name += "-request-advised"
				return nil
			},
		},
	).WithResponseAdvice(goweb.ResponseAdviceFunc(
		func(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error) {
			ctx.Response().Header().Set("X-Controller-Advice", "response-advised")
			return result, nil
		}))
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.message-advice", mvc.NewRestController(
				"advice",
				mvc.POST(
					"/advice/messages",
					mvc.BindJSON(
						http.StatusCreated,
						func(_ *arkweb.Context, input starterAdviceBodyInput) (map[string]string, error) {
							return map[string]string{"name": input.Name}, nil
						},
					),
				),
			)).WithControllerAdvices(advice),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/advice/messages",
			strings.NewReader(`{"name":"goark"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		return request, nil
	}, http.StatusCreated)
	if snapshot.header.Get("X-Controller-Advice") != "response-advised" {
		t.Fatalf(
			"X-Controller-Advice = %q, want response-advised",
			snapshot.header.Get("X-Controller-Advice"),
		)
	}
	if snapshot.body != `{"name":"goark-request-advised"}` {
		t.Fatalf("body = %q, want advised request body", snapshot.body)
	}
}

func TestAutoConfigure_whenRequestBodyAdviceBeanExists_shouldApplyToMVCBindJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(
			gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0"))),
		),
		boot.WithConfiguration(
			starterRequestBodyAdviceConfiguration{},
			mvc.NewConfiguration("test.web.request-body-advice.mvc", mvc.NewRestController(
				"users",
				mvc.POST(
					"/users",
					mvc.BindJSON(
						http.StatusCreated,
						func(_ *arkweb.Context, input starterAdvisedInput) (map[string]string, error) {
							return map[string]string{"name": input.Name}, nil
						},
					),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			starterServerURL(t, app)+"/users",
			strings.NewReader(`{"name":"goark"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", message.MediaTypeJSON)
		request.Header.Set("Accept", message.MediaTypeJSON)
		return request, nil
	}, http.StatusCreated)

	if snapshot.body != `{"name":"goark-advised"}` {
		t.Fatalf("body = %q, want advised JSON body", snapshot.body)
	}
}
func TestAutoConfigure_whenControllerResponseBodyExists_shouldBypassViewResolution(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "status.html"), "<h1>view</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.response-body",
			mvc.NewController(
				"status",
				mvc.GET(
					"/status",
					mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
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

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/status", http.StatusOK)
	if snapshot.header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "UP" {
		t.Fatalf("body = %q, want raw response body", snapshot.body)
	}
}

func (starterRequestBodyAdviceConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return gbcweb.RegisterRequestBodyAdvice(
		config.Registry(),
		"testStarterRequestBodyAdvice",
		message.ReadAdviceFunc{
			After: func(_ *arkweb.Context, input message.ReadAdviceContext) error {
				target, ok := input.Target.(*starterAdvisedInput)
				if ok {
					target.Name += "-advised"
				}
				return nil
			},
		},
		container.WithOrder(-100),
	)
}

func (starterMessageConverterConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return gbcweb.RegisterMessageConverter(
		config.Registry(),
		"testStarterTokenMessageConverter",
		starterTokenConverter{},
		container.WithOrder(-100),
	)
}

type starterLocalePayload struct {
	OK         bool   `json:"ok"`
	Locale     string `json:"locale"`
	Language   string `json:"language"`
	Region     string `json:"region"`
	LocaleSize int    `json:"localeSize"`
}

func (c starterValidatorConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterSearchOwner struct {
	Name  string `form:"name"`
	Level int    `form:"level"`
}

func requestUntilStatus(t *testing.T, target string, statusCode int) string {
	return requestUntilStatusSnapshot(t, target, statusCode).body
}

type starterTokenInput struct {
	Value string
}

func (starterValidatorConfiguration) Name() string {
	return "test.web.validator"
}

const starterWebSocketKey = "dGhlIHNhbXBsZSBub25jZQ=="
