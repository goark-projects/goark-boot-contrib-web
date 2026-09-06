package gbcweb_test

import (
	"bufio"
	"context"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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
	"goark.dev/goark/core/convert"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/static"
)

func TestAutoConfigure_whenControllerAdviceInitBinderExists_shouldUseScopedConvertersThroughArkhos(
	t *testing.T,
) {
	advice := mvc.NewRestControllerAdvice("starter-global-binders").WithInitBinders(
		mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
			return binder.AddConverter(
				mvc.ConverterFunc[string, starterAdviceTenantID](
					func(value string) (starterAdviceTenantID, error) {
						return starterAdviceTenantID{value: "starter-advice:" + value}, nil
					},
				),
			)
		}),
	)
	controller := mvc.NewRestController(
		"starter-advice-search",
		mvc.GET(
			"/advice-binder",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterAdviceSearchPayload, error) {
				criteria, err := mvc.ModelAttribute[starterAdviceSearchCriteria](ctx)
				if err != nil {
					return starterAdviceSearchPayload{}, err
				}
				tenant, err := mvc.RequestParamAs[starterAdviceTenantID](ctx, "tenant")
				if err != nil {
					return starterAdviceSearchPayload{}, err
				}
				return starterAdviceSearchPayload{
					Page:        criteria.Page,
					Tenant:      criteria.Tenant.value,
					ParamTenant: tenant.value,
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
			mvc.NewConfiguration("test.mvc.advice-init-binder", controller).
				WithControllerAdvices(advice),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)
	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/advice-binder?page=goark&tenant=blue",
		http.StatusOK,
	)
	var payload starterAdviceSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("advice init binder json invalid: %v", err)
	}
	if payload.Page != 105 ||
		payload.Tenant != "starter-advice:blue" ||
		payload.ParamTenant != "starter-advice:blue" {
		t.Fatalf("advice init binder payload = %#v, want scoped conversion", payload)
	}
	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	service, err := goark.Get[*convert.Service](
		t.Context(),
		appContext,
		gbcweb.BeanNameConversionService,
	)
	if err != nil {
		t.Fatalf("resolve conversion service failed: %v", err)
	}
	if _, err := convert.Convert[starterAdviceTenantID](service, "blue"); err == nil {
		t.Fatal("starter conversion service should not see advice converter")
	}
}

func TestAutoConfigure_whenMatrixVariablePathScopeExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.matrix-variable", mvc.NewRestController(
				"matrix",
				mvc.GET(
					"/owners/{ownerId}/pets/{petId}",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterMatrixVariablePayload, error) {
							owner, err := mvc.MatrixVariableString(
								ctx,
								"q",
								mvc.WithMatrixPathVariable("ownerId"),
							)
							if err != nil {
								return starterMatrixVariablePayload{}, err
							}
							pet, err := mvc.MatrixVariableString(
								ctx,
								"q",
								mvc.WithMatrixPathVariable("petId"),
							)
							if err != nil {
								return starterMatrixVariablePayload{}, err
							}
							color, err := mvc.MatrixVariableString(
								ctx,
								"color",
								mvc.WithMatrixPathVariable("petId"),
							)
							if err != nil {
								return starterMatrixVariablePayload{}, err
							}
							return starterMatrixVariablePayload{
								Owner: owner,
								Pet:   pet,
								Color: color,
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
	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/owners/42;q=owner/pets/21;q=pet;color=black",
		http.StatusOK,
	)
	var got starterMatrixVariablePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got != (starterMatrixVariablePayload{Owner: "owner", Pet: "pet", Color: "black"}) {
		t.Fatalf("payload = %#v, want path-scoped matrix variables", got)
	}
}

func TestAutoConfigure_whenRestControllerAdviceExists_shouldMapErrors(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.advice-controller", mvc.NewRestController(
				"users",
				mvc.GET(
					"/advice/{id}",
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
				"test.mvc.rest-advice",
				mvc.ExceptionReturnAs[*starterAdviceError](
					http.StatusConflict,
					func(_ *arkweb.Context, err *starterAdviceError) map[string]string {
						return map[string]string{"id": err.id}
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
		starterServerURL(t, app)+"/advice/42",
		http.StatusConflict,
	)
	if snapshot.header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != `{"id":"42"}` {
		t.Fatalf("body = %q, want advice payload", snapshot.body)
	}
}
func (starterWebFeaturesConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	if err := goweb.RegisterInterceptor(
		config.Registry(), "testInterceptor", goweb.InterceptorFunc(
			func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
				ctx.Response().Header().Set("X-Starter-Interceptor", "hit")
				return next.Handle(ctx)
			})); err != nil {
		return err
	}
	if err := goweb.RegisterFilter(
		config.Registry(), "testFilter", servlet.FilterFunc(
			func(
				ctx context.Context,
				req *servlet.Request,
				res servlet.Response,
				chain servlet.Chain,
			) error {
				res.Header().Set("X-Starter-Filter", "hit")
				return chain.Next(ctx, req, res)
			})); err != nil {
		return err
	}
	interceptorMapping, err := goweb.NewInterceptorMapping(
		goweb.WithInterceptorPathPatterns("/**/contracts"),
	)
	if err != nil {
		return err
	}
	if err := goweb.RegisterMappedInterceptor(
		config.Registry(), "testScopedInterceptor", goweb.InterceptorFunc(
			func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
				ctx.Response().Header().Set("X-Starter-Scoped-Interceptor", "hit")
				return next.Handle(ctx)
			}), interceptorMapping); err != nil {
		return err
	}
	filterMapping, err := goweb.NewFilterMapping(goweb.WithFilterPathPatterns("/**/contracts"))
	if err != nil {
		return err
	}
	if err := goweb.RegisterMappedFilter(
		config.Registry(), "testScopedFilter", servlet.FilterFunc(
			func(
				ctx context.Context,
				req *servlet.Request,
				res servlet.Response,
				chain servlet.Chain,
			) error {
				res.Header().Set("X-Starter-Scoped-Filter", "hit")
				return chain.Next(ctx, req, res)
			}), filterMapping); err != nil {
		return err
	}
	return static.Register(config.Registry(), "testStaticResources", "/assets/*", fstest.MapFS{
		"app.txt": &fstest.MapFile{
			Data:    []byte("starter static"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	})
}

func readHandshakeHeaders(t *testing.T, reader *bufio.Reader) http.Header {
	t.Helper()
	headers := http.Header{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read handshake header failed: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return headers
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("invalid handshake header line %q", line)
		}
		headers.Add(textproto.CanonicalMIMEHeaderKey(name), strings.TrimSpace(value))
	}
}

func assertConditionDispatchPayload(t *testing.T, target string, wantMode string) {
	t.Helper()
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	}, http.StatusOK)
	var payload map[string]string
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["mode"] != wantMode {
		t.Fatalf("payload = %#v, want mode %q", payload, wantMode)
	}
}

func (c starterErrorMapperConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type starterIndexedSearchCriteria struct {
	Owners []starterIndexedSearchOwner `form:"owners"`
	Page   int                         `form:"page"`
}

type starterAdvisedInput struct {
	Name string `json:"name"`
}

type starterOptionalBodyInput struct {
	Name string `json:"name" arkarta:"required"`
}

type starterValidatorConfiguration struct{}
