package gbcweb_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	goweb "goark.dev/goark/web"
	weblocale "goark.dev/goark/web/locale"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenControllerInitBinderExists_shouldUseScopedConvertersThroughArkhos(
	t *testing.T,
) {
	localController := mvc.NewRestController(
		"starter-local-search",
		mvc.GET(
			"/init-binder/local",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterScopedSearchPayload, error) {
				criteria, err := mvc.ModelAttribute[starterScopedSearchCriteria](ctx)
				if err != nil {
					return starterScopedSearchPayload{}, err
				}
				tenant, err := mvc.RequestParamAs[starterScopedTenantID](ctx, "tenant")
				if err != nil {
					return starterScopedSearchPayload{}, err
				}
				return starterScopedSearchPayload{
					Page:        criteria.Page,
					Tenant:      criteria.Tenant.value,
					ParamTenant: tenant.value,
				}, nil
			}),
		),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.AddConverter(
			mvc.ConverterFunc[string, starterScopedTenantID](
				func(value string) (starterScopedTenantID, error) {
					return starterScopedTenantID{value: "starter-local:" + value}, nil
				},
			),
		)
	}))
	plainController := mvc.NewRestController(
		"starter-plain-search",
		mvc.GET(
			"/init-binder/plain",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterScopedSearchPayload, error) {
				criteria, err := mvc.ModelAttribute[starterScopedSearchCriteria](ctx)
				if err != nil {
					return starterScopedSearchPayload{}, err
				}
				return starterScopedSearchPayload{
					Page:   criteria.Page,
					Tenant: criteria.Tenant.value,
				}, nil
			}),
		),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterConversionConfiguration{},
			mvc.NewConfiguration("test.mvc.init-binder", localController, plainController),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	localSnapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/init-binder/local?page=goark&tenant=blue",
		http.StatusOK,
	)
	var localPayload starterScopedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(localSnapshot.body), &localPayload); err != nil {
		t.Fatalf("local init binder json invalid: %v", err)
	}
	if localPayload.Page != 105 ||
		localPayload.Tenant != "starter-local:blue" ||
		localPayload.ParamTenant != "starter-local:blue" {
		t.Fatalf("local init binder payload = %#v, want scoped conversion", localPayload)
	}

	requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/init-binder/plain?page=goark&tenant=blue",
		http.StatusBadRequest,
	)
}
func TestAutoConfigure_whenRequestMappingWithoutMethodExists_shouldServeDefaultMethodsThroughArkhos(
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
			mvc.NewConfiguration("test.mvc.request-mapping", mvc.NewRestController(
				"requestMapping",
				mvc.RequestMapping(
					"/request-mapping",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterRequestMappingPayload, error) {
							return starterRequestMappingPayload{
								Method: ctx.Request().Method(),
								Path:   ctx.Request().Path(),
							}, nil
						},
					),
				)...,
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
		snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
			return http.NewRequestWithContext(
				t.Context(),
				method,
				serverURL+"/request-mapping",
				nil,
			)
		}, http.StatusOK)
		var payload starterRequestMappingPayload
		if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
			t.Fatalf("%s response json invalid: %v", method, err)
		}
		if payload.Method != method || payload.Path != "/request-mapping" {
			t.Fatalf("%s payload = %#v, want method and path", method, payload)
		}
	}

	requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(
			t.Context(),
			http.MethodTrace,
			serverURL+"/request-mapping",
			nil,
		)
	}, http.StatusMethodNotAllowed)
}

func TestAutoConfigure_whenSuppressedFieldsExist_shouldExposeBindingResultThroughArkhos(
	t *testing.T,
) {
	controller := mvc.NewRestController(
		"starter-suppressed-fields",
		mvc.GET(
			"/field-binder/suppressed",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterSuppressedPayload, error) {
				input, result, err := mvc.ModelAttributeResult[starterBinderInput](ctx)
				if err != nil {
					return starterSuppressedPayload{}, err
				}
				return starterSuppressedPayload{
					Name:             input.Name,
					Admin:            input.Admin,
					SuppressedFields: result.SuppressedFields(),
				}, nil
			}),
		),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.SetAllowedFields("name")
	}))
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.suppressed-fields", controller),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(
			t,
			app,
		)+"/field-binder/suppressed?name=ada&admin=true&profile.email=ada@example.test",
		http.StatusOK,
	)
	var payload starterSuppressedPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("suppressed fields json invalid: %v", err)
	}
	if payload.Name != "ada" ||
		payload.Admin ||
		len(payload.SuppressedFields) != 2 ||
		payload.SuppressedFields[0] != "admin" ||
		payload.SuppressedFields[1] != "profile.email" {
		t.Fatalf("payload = %#v, want suppressed binding fields", payload)
	}
}
func TestAutoConfigure_whenMVCViewControllerExists_shouldRenderTemplate(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "ready.html"), "<h1>ready</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.view-controller", mvc.NewController(
				"viewControllers",
				mvc.ViewController(
					"/ready",
					"ready",
					mvc.WithViewControllerStatus(http.StatusAccepted),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)
	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/ready",
		http.StatusAccepted,
	)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>ready</h1>" {
		t.Fatalf("view body = %q, want rendered html", snapshot.body)
	}
}
func (starterAttributeConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return goweb.RegisterFilter(
		config.Registry(),
		"testTypedAttributeFilter",
		servlet.FilterFunc(
			func(
				ctx context.Context,
				req *servlet.Request,
				res servlet.Response,
				chain servlet.Chain,
			) error {
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
			},
		),
	)
}

func (starterLocaleChangeConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	interceptor, err := weblocale.ChangeInterceptor(weblocale.WithParameterName("lang"))
	if err != nil {
		return err
	}
	return goweb.RegisterInterceptor(config.Registry(), "testLocaleChangeInterceptor", interceptor)
}

type starterModelAttributeBindingPayload struct {
	Valid        bool   `json:"valid"`
	BindingError bool   `json:"bindingError"`
	Field        string `json:"field"`
	Name         string `json:"name"`
	Page         int    `json:"page"`
}

type starterAdviceSearchPayload struct {
	Page        int    `json:"page"`
	Tenant      string `json:"tenant"`
	ParamTenant string `json:"paramTenant"`
}

type starterNestedSearchCriteria struct {
	Owner *starterSearchOwner `form:"owner"`
	Page  int                 `form:"page"`
}

func requestUntilStatusSnapshot(t *testing.T, target string, statusCode int) responseSnapshot {
	return requestUntilStatusSnapshotWithMethod(t, http.MethodGet, target, statusCode)
}

type starterTokenOutput struct {
	Value string
}

func (starterValidatorConfiguration) Order() int {
	return 0
}

type starterWebSocketConfiguration struct{}
