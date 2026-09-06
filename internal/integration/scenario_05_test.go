package gbcweb_test

import (
	"context"
	"errors"
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

func TestAutoConfigure_whenInitBinderFieldFiltersExist_shouldBindModelAttributesThroughArkhos(
	t *testing.T,
) {
	allowedController := mvc.NewRestController(
		"starter-field-allowed",
		mvc.GET(
			"/field-binder/allowed",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterBinderPayload, error) {
				input, err := mvc.ModelAttribute[starterBinderInput](ctx)
				if err != nil {
					return starterBinderPayload{}, err
				}
				return starterBinderPayloadFromInput(input), nil
			}),
		),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.SetAllowedFields(
			"name",
			"profile.email",
			"roles[*].name",
			"metadata[department]",
		)
	}))
	disallowedController := mvc.NewRestController(
		"starter-field-disallowed",
		mvc.GET(
			"/field-binder/disallowed",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterBinderPayload, error) {
				input, err := mvc.ModelAttribute[starterBinderInput](ctx)
				if err != nil {
					return starterBinderPayload{}, err
				}
				return starterBinderPayloadFromInput(input), nil
			}),
		),
	)
	advice := mvc.NewRestControllerAdvice("starter-field-global-binder").WithInitBinders(
		mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
			return binder.SetDisallowedFields("SYSTEM")
		}),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.field-binders", allowedController, disallowedController).
				WithControllerAdvices(advice),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	baseURL := starterServerURL(t, app)
	query := "?name=ada&system=root&admin=true&profile.email=ada@example.test&profile.admin=true&" +
		"roles[0].name=reader&roles[0].admin=true&metadata[department]=engineering&metadata[secret]=root"

	allowedSnapshot := requestUntilStatusSnapshot(
		t,
		baseURL+"/field-binder/allowed"+query,
		http.StatusOK,
	)
	var allowedPayload starterBinderPayload
	if err := arkjson.Unmarshal(nil, []byte(allowedSnapshot.body), &allowedPayload); err != nil {
		t.Fatalf("allowed field binder json invalid: %v", err)
	}
	if allowedPayload.Name != "ada" ||
		allowedPayload.System != "" ||
		allowedPayload.Admin ||
		allowedPayload.ProfileEmail != "ada@example.test" ||
		allowedPayload.ProfileAdmin {
		t.Fatalf("allowed payload = %#v, want only explicitly allowed fields", allowedPayload)
	}
	if len(allowedPayload.Roles) != 1 || allowedPayload.Roles[0].Name != "reader" ||
		allowedPayload.Roles[0].Admin {
		t.Fatalf("allowed roles = %#v, want only role name", allowedPayload.Roles)
	}
	if allowedPayload.Metadata["department"] != "engineering" {
		t.Fatalf("allowed metadata = %#v, want department", allowedPayload.Metadata)
	}
	if _, ok := allowedPayload.Metadata["secret"]; ok {
		t.Fatalf("allowed metadata = %#v, want secret skipped", allowedPayload.Metadata)
	}

	disallowedSnapshot := requestUntilStatusSnapshot(
		t,
		baseURL+"/field-binder/disallowed"+query,
		http.StatusOK,
	)
	var disallowedPayload starterBinderPayload
	if err := arkjson.Unmarshal(nil, []byte(disallowedSnapshot.body), &disallowedPayload); err != nil {
		t.Fatalf("disallowed field binder json invalid: %v", err)
	}
	if disallowedPayload.Name != "ada" ||
		disallowedPayload.System != "" ||
		!disallowedPayload.Admin ||
		disallowedPayload.ProfileEmail != "ada@example.test" ||
		!disallowedPayload.ProfileAdmin {
		t.Fatalf(
			"disallowed payload = %#v, want only globally disallowed system field skipped",
			disallowedPayload,
		)
	}
	if len(disallowedPayload.Roles) != 1 || disallowedPayload.Roles[0].Name != "reader" ||
		!disallowedPayload.Roles[0].Admin {
		t.Fatalf(
			"disallowed roles = %#v, want role fields except global system",
			disallowedPayload.Roles,
		)
	}
	if disallowedPayload.Metadata["department"] != "engineering" ||
		disallowedPayload.Metadata["secret"] != "root" {
		t.Fatalf(
			"disallowed metadata = %#v, want non-system metadata preserved",
			disallowedPayload.Metadata,
		)
	}
}

func TestAutoConfigure_whenBindingResultRouteExists_shouldHandleValidationResult(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterValidatorConfiguration{},
			mvc.NewConfiguration("test.mvc.binding-result", mvc.NewRestController("bindingResult",
				mvc.POST("/binding-result", mvc.BindJSONResult(
					http.StatusOK,
					func(
						_ *arkweb.Context,
						input starterValidatorRequest,
						result mvc.BindingResult,
					) (starterBindingResultPayload, error) {
						field, ok := result.FieldError("name")
						if !ok {
							return starterBindingResultPayload{
								Valid: result.Valid(),
								Name:  input.Name,
							}, nil
						}
						return starterBindingResultPayload{
							Valid: result.Valid(),
							Path:  field.Path(),
							Code:  field.Code(),
							Name:  input.Name,
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

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			starterServerURL(t, app)+"/binding-result",
			strings.NewReader(`{"name":"root"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)

	var payload starterBindingResultPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("binding result json invalid: %v", err)
	}
	if payload.Valid || payload.Path != "name" || payload.Code != "reserved" ||
		payload.Name != "root" {
		t.Fatalf("binding result payload = %#v, want controller-handled validation result", payload)
	}
}

func TestAutoConfigure_whenFormContentFilterEnabled_shouldBindDeleteFormParameters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  mvc:
    formcontent:
      filter:
        enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.form-content", mvc.NewRestController(
				"items",
				mvc.DELETE(
					"/items/1",
					mvc.Return(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
						return mvc.RequestParamString(ctx, "name")
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
			http.MethodDelete,
			starterServerURL(t, app)+"/items/1",
			strings.NewReader("name=goark"),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return request, nil
	}, http.StatusOK)
	if snapshot.body != "goark" {
		t.Fatalf("body = %q, want form parameter", snapshot.body)
	}
}
func TestAutoConfigure_whenDefaultResourceStaticExists_shouldServeStaticResources(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	staticDir := filepath.Join(resource, "static")
	publicDir := filepath.Join(resource, "public")
	mkdir(t, staticDir)
	mkdir(t, publicDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(staticDir, "app.txt"), "resource static")
	writeFile(t, filepath.Join(publicDir, "public.txt"), "resource public")
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
	if body := requestUntilOK(t, serverURL+"/static/app.txt"); body != "resource static" {
		t.Fatalf("static body = %q, want resource static", body)
	}
	if body := requestUntilOK(t, serverURL+"/static/public.txt"); body != "resource public" {
		t.Fatalf("public static body = %q, want resource public", body)
	}
}

func (starterErrorMapperConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return goweb.RegisterErrorMapper(
		config.Registry(),
		"testErrorMapper",
		goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
			if errors.Is(err, errStarterMapped) {
				return arkweb.Text(http.StatusConflict, "mapped")
			}
			return nil
		}),
	)
}

type starterBinderInput struct {
	Name     string                `form:"name"     json:"name"`
	System   string                `form:"system"   json:"system"`
	Admin    bool                  `form:"admin"    json:"admin"`
	Profile  *starterBinderProfile `form:"profile"  json:"profile"`
	Roles    []starterBinderRole   `form:"roles"    json:"roles"`
	Metadata map[string]string     `form:"metadata" json:"metadata"`
}

func (c starterRequestBodyAdviceConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterRequestPartsPayload struct {
	Names         []string `json:"names"`
	Bodies        []string `json:"bodies"`
	OptionalCount int      `json:"optionalCount"`
}

type starterRequestPartMetadata struct {
	Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" arkarta:"required"`
}

type starterArrayProfile struct {
	Aliases []string `form:"aliases" json:"aliases"`
}

type starterAdviceBodyInput struct {
	Name string `json:"name"`
}

type starterConversionConfiguration struct{}
