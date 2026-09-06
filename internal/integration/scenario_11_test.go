package gbcweb_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/arkarta/websocket/frame"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/core/convert"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/problem"
)

func TestAutoConfigure_whenControllerRequestConditionsExist_shouldServeThroughArkhos(t *testing.T) {
	const routeMediaType = "application/vnd.goark.condition+json"
	const acceptHeader = routeMediaType + ", " + problem.MediaType

	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	controller := mvc.NewRestController(
		"conditions",
		mvc.POST(
			"/conditions",
			mvc.JSON(
				http.StatusCreated,
				func(ctx *arkweb.Context) (starterRequestConditionPayload, error) {
					produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
					return starterRequestConditionPayload{Produces: produces.(string)}, nil
				},
			),
			mvc.WithConsumes(routeMediaType),
			mvc.WithProduces(routeMediaType),
			mvc.WithParams("mode=fast"),
			mvc.WithHeaders("X-Route=enabled"),
		),
	).
		WithConsumes("application/json").
		WithProduces("application/json").
		WithParams("tenant=admin").
		WithHeaders("X-Tenant=admin")

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-conditions", controller)),
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
			serverURL+"/conditions?tenant=admin&mode=fast",
			strings.NewReader("{}"),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", routeMediaType)
		request.Header.Set("Accept", acceptHeader)
		request.Header.Set("X-Tenant", "admin")
		request.Header.Set("X-Route", "enabled")
		return request, nil
	}, http.StatusCreated)
	if got := snapshot.header.Get("Content-Type"); got != routeMediaType {
		t.Fatalf("Content-Type = %q, want route media type", got)
	}
	var payload starterRequestConditionPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Produces != routeMediaType {
		t.Fatalf("payload = %#v, want route produces media type", payload)
	}

	requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/conditions?mode=fast",
			strings.NewReader("{}"),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", routeMediaType)
		request.Header.Set("Accept", acceptHeader)
		request.Header.Set("X-Tenant", "admin")
		request.Header.Set("X-Route", "enabled")
		return request, nil
	}, http.StatusBadRequest)
}

func TestAutoConfigure_whenMVCHandlerInterceptorsExist_shouldServeThroughArkhos(t *testing.T) {
	mapping, err := goweb.NewInterceptorMapping(
		goweb.WithInterceptorPathPatterns("/api/**"),
		goweb.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	configuration := mvc.NewConfiguration(
		"test.mvc.handler-interceptors", mvc.NewRestController("accounts",
		mvc.GET("/api/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "accounts", nil
		})),
		mvc.GET("/api/public/ping", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "pong", nil
		})),
	),
	).
		WithHandlerInterceptors(mvc.HandlerInterceptorFuncs{
			PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
				ctx.Response().Header().Set("X-MVC-Handler-Interceptor", "global")
				return true, nil
			},
		}).
		WithMappedHandlerInterceptor(mvc.HandlerInterceptorFuncs{
			PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
				ctx.Response().Header().Set("X-MVC-Mapped-Handler-Interceptor", "api")
				return true, nil
			},
		}, mapping)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(configuration),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	matched := requestUntilStatusSnapshot(t, serverURL+"/api/accounts", http.StatusOK)
	if matched.header.Get("X-MVC-Handler-Interceptor") != "global" ||
		matched.header.Get("X-MVC-Mapped-Handler-Interceptor") != "api" {
		t.Fatalf("matched headers = %#v, want global and mapped interceptors", matched.header)
	}
	if matched.body != "accounts" {
		t.Fatalf("matched body = %q, want accounts", matched.body)
	}
	excluded := requestUntilStatusSnapshot(t, serverURL+"/api/public/ping", http.StatusOK)
	if excluded.header.Get("X-MVC-Handler-Interceptor") != "global" {
		t.Fatalf(
			"excluded global header = %q, want global",
			excluded.header.Get("X-MVC-Handler-Interceptor"),
		)
	}
	if got := excluded.header.Get("X-MVC-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("excluded mapped header = %q, want empty", got)
	}
	if excluded.body != "pong" {
		t.Fatalf("excluded body = %q, want pong", excluded.body)
	}
}

func TestAutoConfigure_whenResponseAdviceBeanExists_shouldApplyToMVCResponseBody(t *testing.T) {
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
			starterResponseAdviceConfiguration{},
			mvc.NewConfiguration("test.web.response-advice.mvc", mvc.NewRestController(
				"profiles",
				mvc.GET(
					"/profiles/current",
					mvc.ResponseBody(
						http.StatusOK,
						func(_ *arkweb.Context) (map[string]string, error) {
							return map[string]string{"name": "goark"}, nil
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
			starterServerURL(t, app)+"/profiles/current",
			nil,
		)
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

func TestAutoConfigure_whenSamePathRequestConditionsExist_shouldDispatchThroughArkhos(
	t *testing.T,
) {
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
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.request-condition-dispatch", mvc.NewRestController(
				"conditionDispatch",
				mvc.GET(
					"/condition-dispatch",
					mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
						return map[string]string{"mode": "fast"}, nil
					}),
					mvc.WithParams("mode=fast"),
				),
				mvc.GET(
					"/condition-dispatch",
					mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
						return map[string]string{"mode": "default"}, nil
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
	assertConditionDispatchPayload(t, serverURL+"/condition-dispatch?mode=fast", "fast")
	assertConditionDispatchPayload(t, serverURL+"/condition-dispatch", "default")
}

func (starterConversionConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	if err := gbcweb.RegisterConverter(
		config.Registry(), "testStringIntConverter",
		convert.ConverterFunc[string, int](func(value string) (int, error) {
		return len(value) + 100, nil
	})); err != nil {
		return err
	}
	return gbcweb.RegisterConverter(
		config.Registry(),
		"testStringTenantIDConverter",
		convert.ConverterFunc[string, starterTenantID](func(value string) (starterTenantID, error) {
			return starterTenantID{value: "tenant:" + value}, nil
		}),
	)
}

type starterPreferencePayload struct {
	Theme      string `json:"theme"`
	NotifySet  bool   `json:"notifySet"`
	Notify     bool   `json:"notify"`
	ConfirmSet bool   `json:"confirmSet"`
	Confirm    bool   `json:"confirm"`
	ProfileSet bool   `json:"profileSet"`
	Subscribed bool   `json:"subscribed"`
	TagsNil    bool   `json:"tagsNil"`
	TagsLength int    `json:"tagsLength"`
}

func failHTTPServer(errors chan<- error, writer http.ResponseWriter, format string, args ...any) {
	select {
	case errors <- fmt.Errorf(format, args...):
	default:
	}
	http.Error(writer, "server assertion failed", http.StatusInternalServerError)
}

func writeClientFrame(t *testing.T, conn net.Conn, next frame.Frame) {
	t.Helper()
	if err := frame.Write(conn, next); err != nil {
		t.Fatalf("write websocket frame failed: %v", err)
	}
}

type starterScopedSearchCriteria struct {
	Page   int                   `form:"page"`
	Tenant starterScopedTenantID `form:"tenant"`
}

func (e *starterAdviceError) Error() string {
	return "starter advice " + e.id
}

func (starterLocaleChangeConfiguration) Name() string {
	return "test.web.locale.change"
}

func (starterRejectingValidator) Validate(_ context.Context, _ any) (validation.Result, error) {
	return validation.NewResult(validation.NewViolation("name", "reserved", "名称不可用", nil)), nil
}

var _ goweb.MessageConverter = starterTokenConverter{}
