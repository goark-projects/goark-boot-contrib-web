package gbcweb_test

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
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

func TestAutoConfigure_whenRequestEntityRouteExists_shouldServeThroughArkhos(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.request-entity", mvc.NewRestController("requestEntity",
				mvc.POST("/request-entity/{id}", mvc.BindRequestEntity(
					http.StatusAccepted,
					func(
						ctx *arkweb.Context,
						entity goweb.RequestEntity[starterRequestEntityInput],
					) (starterRequestEntityPayload, error) {
						id, err := mvc.PathString(ctx, "id")
						if err != nil {
							return starterRequestEntityPayload{}, err
						}
						traceID, _ := entity.HeaderValue("X-Trace-ID")
						body, hasBody := entity.Body()
						return starterRequestEntityPayload{
							ID:            id,
							Name:          body.Name,
							Method:        entity.Method(),
							URL:           entity.URL(),
							Path:          entity.Path(),
							TraceID:       traceID,
							ContentLength: entity.ContentLength(),
							HasBody:       hasBody && entity.HasBody(),
						}, nil
					},
				)),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	target := starterServerURL(t, app) + "/request-entity/42?mode=full"
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			target,
			strings.NewReader(`{"name":"goark"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		request.Header.Set("X-Trace-ID", "trace-1")
		return request, nil
	}, http.StatusAccepted)

	var payload starterRequestEntityPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.ID != "42" || payload.Name != "goark" || payload.Method != http.MethodPost ||
		payload.URL != target || payload.Path != "/request-entity/42" || payload.TraceID != "trace-1" ||
		payload.ContentLength != int64(len(`{"name":"goark"}`)) || !payload.HasBody {
		t.Fatalf("request entity payload = %#v, want metadata and body", payload)
	}
}

func TestAutoConfigure_whenFieldPrefixesExist_shouldBindModelAttributesThroughArkhos(t *testing.T) {
	defaultController := mvc.NewRestController(
		"starter-field-prefix-default",
		mvc.GET(
			"/field-prefix/default",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterPreferencePayload, error) {
				input, err := mvc.ModelAttribute[starterPreferenceInput](ctx)
				if err != nil {
					return starterPreferencePayload{}, err
				}
				return starterPreferencePayloadFromInput(input), nil
			}),
		),
	)
	customController := mvc.NewRestController(
		"starter-field-prefix-custom",
		mvc.GET(
			"/field-prefix/custom",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterPreferencePayload, error) {
				input, err := mvc.ModelAttribute[starterPreferenceInput](ctx)
				if err != nil {
					return starterPreferencePayload{}, err
				}
				return starterPreferencePayloadFromInput(input), nil
			}),
		),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		if err := binder.SetFieldDefaultPrefix("~"); err != nil {
			return err
		}
		return binder.SetFieldMarkerPrefix("__")
	}))
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.field-prefixes", defaultController, customController),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	baseURL := starterServerURL(t, app)
	defaultSnapshot := requestUntilStatusSnapshot(
		t,
		baseURL+"/field-prefix/default?!theme=dark&_notify=on&"+
			"confirm=true&_confirm=on&_profile.subscribed=on&_tags=on",
		http.StatusOK,
	)
	var defaultPayload starterPreferencePayload
	if err := arkjson.Unmarshal(nil, []byte(defaultSnapshot.body), &defaultPayload); err != nil {
		t.Fatalf("default field prefix json invalid: %v", err)
	}
	assertStarterPreferencePayload(t, defaultPayload)

	customSnapshot := requestUntilStatusSnapshot(
		t,
		baseURL+"/field-prefix/custom?~theme=dark&__notify=on&"+
			"confirm=true&__confirm=on&__profile.subscribed=on&__tags=on",
		http.StatusOK,
	)
	var customPayload starterPreferencePayload
	if err := arkjson.Unmarshal(nil, []byte(customSnapshot.body), &customPayload); err != nil {
		t.Fatalf("custom field prefix json invalid: %v", err)
	}
	assertStarterPreferencePayload(t, customPayload)
}

func TestAutoConfigure_whenControllerAdviceValueExists_shouldRenderTemplate(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "missing.html"), "<h1>missing</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.advice-view", mvc.NewController(
				"pages",
				mvc.GET(
					"/pages/missing",
					mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
						return nil, &starterAdviceError{id: "missing"}
					}),
				),
			)),
			mvc.NewControllerAdvice(
				"test.mvc.page-advice",
				mvc.ExceptionReturnAs[*starterAdviceError](
					http.StatusNotFound,
					func(_ *arkweb.Context, _ *starterAdviceError) string {
						return "missing"
					},
				),
			),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/pages/missing",
		http.StatusNotFound,
	)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>missing</h1>" {
		t.Fatalf("body = %q, want rendered advice view", snapshot.body)
	}
}

func TestAutoConfigure_whenWebErrorMapperConfigurerExists_shouldApplyMapper(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc", mvc.NewController(
				"errors",
				mvc.GET(
					"/errors",
					mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
						return nil, errStarterMapped
					}),
				),
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
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](
		t.Context(),
		appContext,
		gbcarkhos.BeanNameServer,
	)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	body := requestUntilStatus(t, server.URL()+"/errors", http.StatusConflict)
	if body != "mapped" {
		t.Fatalf("body = %q, want mapped", body)
	}
}

func requestStarterSessionAttributesPayload(
	t *testing.T,
	serverURL string,
	cookieHeader string,
) starterSessionAttributesPayload {
	t.Helper()

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/wizard/current",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Cookie", cookieHeader)
		return request, nil
	}, http.StatusOK)
	var payload starterSessionAttributesPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	return payload
}

type starterIndexedSearchPayload struct {
	FirstName   string `json:"firstName"`
	FirstLevel  int    `json:"firstLevel"`
	FirstAlias  string `json:"firstAlias"`
	SecondName  string `json:"secondName"`
	SecondLevel int    `json:"secondLevel"`
	SecondAlias string `json:"secondAlias"`
	Page        int    `json:"page"`
}

func (c starterHTTPClientCustomizerConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (c starterWebFeaturesConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterBinderRole struct {
	Name  string `form:"name"  json:"name"`
	Admin bool   `form:"admin" json:"admin"`
}

func (starterErrorMapperConfiguration) Order() int {
	return 0
}

func (starterMessageConverterConfiguration) Order() int {
	return 0
}

type starterPathPrefixPayload struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}
