package gbcweb_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
)

const starterTokenMediaType = "application/vnd.goark.starter-token"

type starterTokenInput struct {
	Value string
}

type starterTokenOutput struct {
	Value string
}

type starterAdvisedInput struct {
	Name string `json:"name"`
}

type starterTokenConverter struct{}

func (starterTokenConverter) MediaTypes() []string {
	return []string{starterTokenMediaType}
}

func (starterTokenConverter) CanRead(target any, mediaType string) bool {
	_, ok := target.(*starterTokenInput)
	return ok && strings.HasPrefix(mediaType, starterTokenMediaType)
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

func (starterTokenConverter) CanWrite(value any, mediaType string) bool {
	_, ok := value.(starterTokenOutput)
	return ok && strings.HasPrefix(mediaType, starterTokenMediaType)
}

func (starterTokenConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	output := value.(starterTokenOutput)
	if err := servlet.SetContentType(ctx.Response(), mediaType); err != nil {
		return err
	}
	_, err := ctx.Response().WriteString("starter:" + output.Value)
	return err
}

func TestAutoConfigure_whenMessageConverterBeanExists_shouldUseItForMVCBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")))),
		boot.WithConfiguration(
			starterMessageConverterConfiguration{},
			mvc.NewConfiguration("test.web.message-converter.mvc", mvc.NewController("tokens",
				mvc.POST("/tokens", mvc.BindBody(http.StatusCreated, func(_ *arkweb.Context, input starterTokenInput) (starterTokenOutput, error) {
					return starterTokenOutput{Value: input.Value}, nil
				}), mvc.WithConsumes(starterTokenMediaType), mvc.WithProduces(starterTokenMediaType)),
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
	if _, err := goark.Get[message.Reader](t.Context(), appContext, gbcweb.BeanNameMessageReader); err != nil {
		t.Fatalf("resolve message reader failed: %v", err)
	}
	if _, err := goark.Get[message.Writer](t.Context(), appContext, gbcweb.BeanNameMessageWriter); err != nil {
		t.Fatalf("resolve message writer failed: %v", err)
	}

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/tokens", strings.NewReader("starter=abc"))
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

func TestAutoConfigure_whenRequestBodyAdviceBeanExists_shouldApplyToMVCBindJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")))),
		boot.WithConfiguration(
			starterRequestBodyAdviceConfiguration{},
			mvc.NewConfiguration("test.web.request-body-advice.mvc", mvc.NewRestController("users",
				mvc.POST("/users", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input starterAdvisedInput) (map[string]string, error) {
					return map[string]string{"name": input.Name}, nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/users", strings.NewReader(`{"name":"goark"}`))
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

func TestAutoConfigure_whenResponseAdviceBeanExists_shouldApplyToMVCResponseBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")))),
		boot.WithConfiguration(
			starterResponseAdviceConfiguration{},
			mvc.NewConfiguration("test.web.response-advice.mvc", mvc.NewRestController("profiles",
				mvc.GET("/profiles/current", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return map[string]string{"name": "goark"}, nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, starterServerURL(t, app)+"/profiles/current", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", message.MediaTypeJSON)
		return request, nil
	}, http.StatusAccepted)

	if snapshot.body != `{"name":"goark-advised"}` {
		t.Fatalf("body = %q, want advised JSON body", snapshot.body)
	}
}

func TestAutoConfigure_whenFormBodyExists_shouldUseDefaultMessageConverters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")))),
		boot.WithConfiguration(mvc.NewConfiguration("test.web.form-message.mvc", mvc.NewController("forms",
			mvc.POST("/forms", mvc.BindBody(http.StatusAccepted, func(_ *arkweb.Context, input url.Values) (url.Values, error) {
				input.Set("seen", "true")
				return input, nil
			}), mvc.WithConsumes(message.MediaTypeFormURLEncoded), mvc.WithProduces(message.MediaTypeFormURLEncoded)),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/forms", strings.NewReader("name=goark&tag=web&tag=mvc"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", message.MediaTypeFormURLEncoded)
		request.Header.Set("Accept", message.MediaTypeFormURLEncoded)
		return request, nil
	}, http.StatusAccepted)

	if got := snapshot.header.Get("Content-Type"); got != message.MediaTypeFormURLEncoded {
		t.Fatalf("Content-Type = %q, want form", got)
	}
	if snapshot.body != "name=goark&seen=true&tag=web&tag=mvc" {
		t.Fatalf("body = %q, want encoded form", snapshot.body)
	}
}

type starterMessageConverterConfiguration struct{}

func (starterMessageConverterConfiguration) Name() string {
	return "test.web.message-converter"
}

func (starterMessageConverterConfiguration) Order() int {
	return 0
}

func (c starterMessageConverterConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterMessageConverterConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return gbcweb.RegisterMessageConverter(config.Registry(), "testStarterTokenMessageConverter", starterTokenConverter{}, container.WithOrder(-100))
}

type starterRequestBodyAdviceConfiguration struct{}

func (starterRequestBodyAdviceConfiguration) Name() string {
	return "test.web.request-body-advice"
}

func (starterRequestBodyAdviceConfiguration) Order() int {
	return 0
}

func (c starterRequestBodyAdviceConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterRequestBodyAdviceConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return gbcweb.RegisterRequestBodyAdvice(config.Registry(), "testStarterRequestBodyAdvice", message.ReadAdviceFunc{
		After: func(_ *arkweb.Context, input message.ReadAdviceContext) error {
			target, ok := input.Target.(*starterAdvisedInput)
			if ok {
				target.Name += "-advised"
			}
			return nil
		},
	}, container.WithOrder(-100))
}

type starterResponseAdviceConfiguration struct{}

func (starterResponseAdviceConfiguration) Name() string {
	return "test.web.response-advice"
}

func (starterResponseAdviceConfiguration) Order() int {
	return 0
}

func (c starterResponseAdviceConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterResponseAdviceConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return gbcweb.RegisterResponseAdvice(config.Registry(), "testStarterResponseAdvice", arkweb.ResponseAdviceFunc(func(_ *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
		return arkweb.JSON(http.StatusAccepted, map[string]string{"name": "goark-advised"}), nil
	}), container.WithOrder(-100))
}

var _ goweb.MessageConverter = starterTokenConverter{}
