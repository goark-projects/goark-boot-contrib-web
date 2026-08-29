package gbcweb_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/web/mvc"
)

type starterRejectingValidator struct{}

func (starterRejectingValidator) Validate(_ context.Context, _ any) (validation.Result, error) {
	return validation.NewResult(validation.NewViolation("name", "reserved", "名称不可用", nil)), nil
}

type starterValidatorRequest struct {
	Name string `json:"name"`
}

func TestAutoConfigure_whenValidatorExists_shouldBindMVCValidation(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterValidatorConfiguration{},
			mvc.NewConfiguration("test.mvc.validator", mvc.NewController("validator",
				mvc.POST("/validated", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input starterValidatorRequest) (map[string]string, error) {
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
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/validated", strings.NewReader(`{"name":"root"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		return request, nil
	}, http.StatusUnprocessableEntity)
	if !strings.Contains(snapshot.body, `"code":"reserved"`) || !strings.Contains(snapshot.body, `"path":"name"`) {
		t.Fatalf("validation body = %q, want custom violation", snapshot.body)
	}
}

type starterValidatorConfiguration struct{}

func (starterValidatorConfiguration) Name() string {
	return "test.web.validator"
}

func (starterValidatorConfiguration) Order() int {
	return 0
}

func (c starterValidatorConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterValidatorConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return gbcweb.RegisterValidator(config.Registry(), "testRejectingValidator", starterRejectingValidator{})
}
