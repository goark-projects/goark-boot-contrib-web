package gbcweb_test

import (
	"context"
	"errors"
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
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenMessageConverterBeanExists_shouldUseItForMVCBody(t *testing.T) {
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
			starterMessageConverterConfiguration{},
			mvc.NewConfiguration("test.web.message-converter.mvc", mvc.NewController(
				"tokens",
				mvc.POST(
					"/tokens",
					mvc.BindBody(
						http.StatusCreated,
						func(_ *arkweb.Context, input starterTokenInput) (starterTokenOutput, error) {
							return starterTokenOutput(input), nil
						},
					),
					mvc.WithConsumes(starterTokenMediaType),
					mvc.WithProduces(starterTokenMediaType),
				),
			)),
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
	if _, err := goark.Get[message.Reader](
		t.Context(), appContext, gbcweb.BeanNameMessageReader,
	); err != nil {
		t.Fatalf("resolve message reader failed: %v", err)
	}
	if _, err := goark.Get[message.Writer](
		t.Context(), appContext, gbcweb.BeanNameMessageWriter,
	); err != nil {
		t.Fatalf("resolve message writer failed: %v", err)
	}

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			starterServerURL(t, app)+"/tokens",
			strings.NewReader("starter=abc"),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", starterTokenMediaType)
		request.Header.Set("Accept", starterTokenMediaType)
		return request, nil
	}, http.StatusCreated)

	if got := snapshot.header.Get("Content-Type"); got != starterTokenMediaType {
		t.Fatalf("Content-Type = %q, want %s", got, starterTokenMediaType)
	}
	if snapshot.body != "starter:abc" {
		t.Fatalf("body = %q, want starter converter output", snapshot.body)
	}
}

func TestAutoConfigure_whenMVCValidationGroupsExist_shouldUseExplicitGroups(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.validation-groups", mvc.NewRestController(
				"users",
				mvc.POST(
					"/users",
					mvc.BindJSONGroups(
						http.StatusCreated,
						func(_ *arkweb.Context, input starterGroupedCreateRequest) (map[string]string, error) {
							return map[string]string{"name": input.Name}, nil
						},
						"create",
					),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	missingName := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/users",
			strings.NewReader(`{}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		return request, nil
	}, http.StatusUnprocessableEntity)
	if missingName.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want problem json", missingName.header.Get("Content-Type"))
	}

	created := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/users",
			strings.NewReader(`{"name":"goark"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		return request, nil
	}, http.StatusCreated)
	if created.body != `{"name":"goark"}` {
		t.Fatalf("body = %q, want created payload", created.body)
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
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.hidden-method", mvc.NewRestController(
				"items",
				mvc.POST(
					"/items/1",
					mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
						return http.MethodPost, nil
					}),
				),
				mvc.DELETE(
					"/items/1",
					mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
						return http.MethodDelete, nil
					}),
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
			starterServerURL(t, app)+"/items/1",
			strings.NewReader("_method=DELETE"),
		)
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

func TestAutoConfigure_whenControllerReturnsRedirectViewName_shouldWriteRedirect(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.redirect", mvc.NewController(
				"redirects",
				mvc.GET(
					"/accounts",
					mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
						return "redirect:/signin", nil
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
	}, http.StatusFound)
	if got := snapshot.header.Get("Location"); got != "/signin" {
		t.Fatalf("Location = %q, want /signin", got)
	}
	if snapshot.body != "" {
		t.Fatalf("body = %q, want empty", snapshot.body)
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
	assertConfigurationCount(
		t,
		appContext.Configurations(),
		"goark.boot.contrib.web.configuration",
		1,
	)
	assertConfigurationCount(
		t,
		appContext.Configurations(),
		"goark.boot.contrib.arkhos.configuration",
		1,
	)
}

func (starterTokenConverter) Read(ctx *arkweb.Context, target any, _ string) error {
	input := target.(*starterTokenInput)
	data, err := io.ReadAll(ctx.Request().Body())
	if err != nil {
		return err
	}
	input.Value = strings.TrimPrefix(string(data), "starter=")
	return nil
}

func (c starterConversionConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterMatrixVariablePayload struct {
	Owner string `json:"owner"`
	Pet   string `json:"pet"`
	Color string `json:"color"`
}

type starterSearchPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
}

func (starterHTTPClientCustomizerConfiguration) Order() int {
	return 0
}

func (starterResponseAdviceConfiguration) Order() int {
	return 0
}

var errStarterMapped = errors.New("starter mapped")

type starterTokenConverter struct{}
