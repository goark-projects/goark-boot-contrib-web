package gbcweb_test

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
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

type starterAttributeProfile struct {
	ID string `json:"id"`
}

type starterAttributePayload struct {
	TraceID   string `json:"traceId"`
	ProfileID string `json:"profileId"`
	Limit     int    `json:"limit"`
}

func TestAutoConfigure_whenTypedAttributesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterAttributeConfiguration{},
			mvc.NewConfiguration("test.mvc.typed-attributes", mvc.NewRestController("typedAttributes",
				mvc.GET("/attributes/typed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterAttributePayload, error) {
					profile, err := mvc.RequestAttribute[starterAttributeProfile](ctx, "profile")
					if err != nil {
						return starterAttributePayload{}, err
					}
					limit, err := mvc.RequestAttribute[int](ctx, "limit")
					if err != nil {
						return starterAttributePayload{}, err
					}
					traceID, err := mvc.SessionAttribute[string](ctx, "traceID")
					if err != nil {
						return starterAttributePayload{}, err
					}
					return starterAttributePayload{TraceID: traceID, ProfileID: profile.ID, Limit: limit}, nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/attributes/typed", http.StatusOK)
	var payload starterAttributePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if payload.TraceID != "trace-1" || payload.ProfileID != "p-1" || payload.Limit != 42 {
		t.Fatalf("payload = %#v, want typed attributes", payload)
	}
}

type starterAttributeConfiguration struct{}

func (starterAttributeConfiguration) Name() string {
	return "test.web.typed-attributes"
}

func (starterAttributeConfiguration) Order() int {
	return 0
}

func (c starterAttributeConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterAttributeConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return goweb.RegisterFilter(config.Registry(), "testTypedAttributeFilter", servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		req.SetAttribute("profile", starterAttributeProfile{ID: "p-1"})
		req.SetAttribute("limit", "42")
		current, err := session.NewMemoryManager().Create(ctx)
		if err != nil {
			return err
		}
		if err := current.SetAttribute("traceID", "trace-1"); err != nil {
			return err
		}
		req.SetAttribute(session.AttributeCurrentSession, current)
		return chain.Next(ctx, req, res)
	}))
}

type starterModelAttributeSourcesCriteria struct {
	TenantID       string `form:"tenantId" json:"tenantId"`
	UserID         int64  `form:"userId" json:"userId"`
	RequestID      string `form:"xRequestId" json:"requestId"`
	AcceptLanguage string `json:"acceptLanguage"`
	Mode           string `form:"mode" json:"mode"`
}

func TestAutoConfigure_whenModelAttributeSourcesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-attribute-sources", mvc.NewRestController("modelAttributeSources",
			mvc.GET("/tenants/{tenantId}/users/{userId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterModelAttributeSourcesCriteria, error) {
				return mvc.ModelAttribute[starterModelAttributeSourcesCriteria](ctx)
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, starterServerURL(t, app)+"/tenants/core;scope=internal/users/42;role=admin?tenantId=query&mode=query", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("X-Request-Id", "req-1")
		request.Header.Set("Accept-Language", "zh-CN")
		request.Header.Set("Mode", "header")
		return request, nil
	}, http.StatusOK)

	var got starterModelAttributeSourcesCriteria
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got.TenantID != "query" ||
		got.UserID != 42 ||
		got.RequestID != "req-1" ||
		got.AcceptLanguage != "zh-CN" ||
		got.Mode != "query" {
		t.Fatalf("criteria = %#v, want model attribute sources with request parameter priority", got)
	}
}

type starterModelNameAccount struct {
	Name string
}

func TestAutoConfigure_whenModelNameInferred_shouldRenderTemplateThroughArkhos(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates", "accounts")
	mkdir(t, templateDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(templateDir, "detail.html"), "<h1>{{.starterModelNameAccount.Name}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(resource)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-name", mvc.NewController("accounts",
			mvc.GET("/accounts/detail", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
				return mvc.NewModelAndView("accounts/detail", starterModelNameAccount{Name: "Goark"}), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/accounts/detail", http.StatusOK)
	if snapshot.body != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want rendered inferred model", snapshot.body)
	}
}

type starterAdviceBodyInput struct {
	Name string `json:"name"`
}

func TestAutoConfigure_whenControllerAdviceMessageAdviceExists_shouldServeThroughArkhos(t *testing.T) {
	advice := mvc.NewRestControllerAdvice("test.mvc.message-advice").WithRequestBodyAdvice(
		goweb.RequestBodyAdviceFunc{
			After: func(_ *arkweb.Context, input goweb.RequestBodyAdviceContext) error {
				target := input.Target.(*starterAdviceBodyInput)
				target.Name += "-request-advised"
				return nil
			},
		},
	).WithResponseAdvice(goweb.ResponseAdviceFunc(func(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error) {
		ctx.Response().Header().Set("X-Controller-Advice", "response-advised")
		return result, nil
	}))
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.message-advice", mvc.NewRestController("advice",
			mvc.POST("/advice/messages", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input starterAdviceBodyInput) (map[string]string, error) {
				return map[string]string{"name": input.Name}, nil
			})),
		)).WithControllerAdvices(advice)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/advice/messages", strings.NewReader(`{"name":"goark"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		return request, nil
	}, http.StatusCreated)
	if snapshot.header.Get("X-Controller-Advice") != "response-advised" {
		t.Fatalf("X-Controller-Advice = %q, want response-advised", snapshot.header.Get("X-Controller-Advice"))
	}
	if snapshot.body != `{"name":"goark-request-advised"}` {
		t.Fatalf("body = %q, want advised request body", snapshot.body)
	}
}

func TestAutoConfigure_whenConditionalRequestMatches_shouldReturnNotModified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	modified := time.Date(2026, time.August, 29, 8, 30, 0, 900, time.FixedZone("CST", 8*60*60))

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.conditional", mvc.NewController("conditional",
			mvc.GET("/reports/summary", mvc.Handler(func(ctx *arkweb.Context) (arkweb.Result, error) {
				if goweb.CheckNotModified(ctx, "summary-v1", modified) {
					return nil, nil
				}
				return goweb.OK(map[string]string{"state": "fresh"}).
					WithETag("summary-v1").
					WithLastModified(modified), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	first := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/reports/summary", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)
	if first.header.Get("ETag") != `"summary-v1"` ||
		first.header.Get("Last-Modified") != "Sat, 29 Aug 2026 00:30:00 GMT" ||
		first.body != `{"state":"fresh"}` {
		t.Fatalf("initial response = %#v body %q", first.header, first.body)
	}

	second := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/reports/summary", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		request.Header.Set("If-None-Match", first.header.Get("ETag"))
		return request, nil
	}, http.StatusNotModified)
	if second.header.Get("ETag") != `"summary-v1"` ||
		second.header.Get("Last-Modified") != "Sat, 29 Aug 2026 00:30:00 GMT" ||
		second.body != "" {
		t.Fatalf("not modified response = %#v body %q", second.header, second.body)
	}
}
