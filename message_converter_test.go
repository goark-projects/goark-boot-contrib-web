package gbcweb_test

import (
	"context"
	"io"
	"net/http"
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

var _ goweb.MessageConverter = starterTokenConverter{}
