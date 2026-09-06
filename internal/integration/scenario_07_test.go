package gbcweb_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
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
)

func TestAutoConfigure_whenParameterMapsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.parameter-map", mvc.NewRestController(
				"search",
				mvc.GET(
					"/tenants/{tenantId}/search/{userId}",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterParameterMapPayload, error) {
							path, err := mvc.PathVariableMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							params, err := mvc.RequestParamMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							paramValues, err := mvc.RequestParamValuesMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							headers, err := mvc.RequestHeaderMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							headerValues, err := mvc.RequestHeaderValuesMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							cookies, err := mvc.CookieValueMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							cookieValues, err := mvc.CookieValueValuesMap(ctx)
							if err != nil {
								return starterParameterMapPayload{}, err
							}
							return starterParameterMapPayload{
								Path:         path,
								Params:       params,
								ParamValues:  paramValues,
								Headers:      headers,
								HeaderValues: headerValues,
								Cookies:      cookies,
								CookieValues: cookieValues,
							}, nil
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
			http.MethodGet,
			starterServerURL(
				t,
				app,
			)+"/tenants/core;scope=internal/search/42;role=admin?tag=query&tag=web&q=goark",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Add("X-Role", "admin")
		request.Header.Add("X-Role", "ops")
		request.Header.Set("X-Request-ID", "req-1")
		request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
		request.AddCookie(&http.Cookie{Name: "role", Value: "admin"})
		request.AddCookie(&http.Cookie{Name: "role", Value: "ops"})
		return request, nil
	}, http.StatusOK)

	var got starterParameterMapPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Path, map[string]string{"tenantId": "core", "userId": "42"}) {
		t.Fatalf("path = %#v", got.Path)
	}
	if got.Params["tag"] != "query" || got.Params["q"] != "goark" {
		t.Fatalf("params = %#v", got.Params)
	}
	if !reflect.DeepEqual(got.ParamValues["tag"], []string{"query", "web"}) {
		t.Fatalf("param values = %#v", got.ParamValues)
	}
	if got.Headers["X-Role"] != "admin" || got.Headers["X-Request-Id"] != "req-1" {
		t.Fatalf("headers = %#v", got.Headers)
	}
	if !reflect.DeepEqual(got.HeaderValues["X-Role"], []string{"admin", "ops"}) {
		t.Fatalf("header values = %#v", got.HeaderValues)
	}
	if !reflect.DeepEqual(got.Cookies, map[string]string{"theme": "dark", "role": "admin"}) {
		t.Fatalf("cookies = %#v", got.Cookies)
	}
	if !reflect.DeepEqual(got.CookieValues["role"], []string{"admin", "ops"}) ||
		!reflect.DeepEqual(got.CookieValues["theme"], []string{"dark"}) {
		t.Fatalf("cookie values = %#v", got.CookieValues)
	}
}

func TestAutoConfigure_whenFormBodyExists_shouldUseDefaultMessageConverters(t *testing.T) {
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
			mvc.NewConfiguration("test.web.form-message.mvc", mvc.NewController(
				"forms",
				mvc.POST(
					"/forms",
					mvc.BindBody(
						http.StatusAccepted,
						func(_ *arkweb.Context, input url.Values) (url.Values, error) {
							input.Set("seen", "true")
							return input, nil
						},
					),
					mvc.WithConsumes(message.MediaTypeFormURLEncoded),
					mvc.WithProduces(message.MediaTypeFormURLEncoded),
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
			starterServerURL(t, app)+"/forms",
			strings.NewReader("name=goark&tag=web&tag=mvc"),
		)
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

func TestAutoConfigure_whenControllerAdviceResponseEntityExists_shouldPreserveEntity(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.advice-entity", mvc.NewRestController(
				"users",
				mvc.GET(
					"/users/{id}",
					mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
						id, err := mvc.PathString(ctx, "id")
						if err != nil {
							return nil, err
						}
						return nil, &starterAdviceError{id: id}
					}),
				),
			)),
			mvc.NewRestControllerAdvice(
				"test.mvc.entity-advice",
				mvc.ExceptionEntityAs[*starterAdviceError](
					func(_ *arkweb.Context, err *starterAdviceError) goweb.ResponseEntity[map[string]string] {
						return goweb.Status(http.StatusGone, map[string]string{"id": err.id}).
							WithHeader("X-Starter-Advice", "entity")
					},
				),
			),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/users/42", http.StatusGone)
	if snapshot.header.Get("X-Starter-Advice") != "entity" {
		t.Fatalf("X-Starter-Advice = %q, want entity", snapshot.header.Get("X-Starter-Advice"))
	}
	if snapshot.body != `{"id":"42"}` {
		t.Fatalf("body = %q, want advice response entity payload", snapshot.body)
	}
}

func requestUntilStatusWithClient(
	t *testing.T,
	client http.Client,
	build func() (*http.Request, error),
	statusCode int,
) responseSnapshot {
	t.Helper()
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(3 * time.Second)
	for {
		request, err := build()
		if err != nil {
			t.Fatalf("build request failed: %v", err)
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
			t.Fatalf(
				"status = %d, want %d, body = %q",
				response.StatusCode,
				statusCode,
				string(body),
			)
		}
		if time.Now().After(deadline) {
			t.Fatalf("request did not succeed before deadline: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func starterServerURL(t *testing.T, app *boot.Application) string {
	t.Helper()
	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](
		t.Context(),
		appContext,
		gbcarkhos.BeanNameServer,
	)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	return server.URL()
}

func (starterTokenConverter) Write(ctx *arkweb.Context, value any, mediaType string) error {
	output := value.(starterTokenOutput)
	if err := servlet.SetContentType(ctx.Response(), mediaType); err != nil {
		return err
	}
	_, err := ctx.Response().WriteString("starter:" + output.Value)
	return err
}

func (c starterAttributeConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func sessionAttributeString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type starterRequestMappingPayload struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func (starterConversionConfiguration) Name() string {
	return "test.web.conversion"
}

func (starterWebSocketConfiguration) Order() int {
	return 0
}

const starterTokenMediaType = "application/vnd.goark.starter-token"
