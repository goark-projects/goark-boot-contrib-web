package gbcweb_test

import (
	"context"
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenControllerAdviceInitBinderExists_shouldUseScopedConvertersThroughArkhos(t *testing.T) {
	advice := mvc.NewRestControllerAdvice("starter-global-binders").WithInitBinders(
		mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
			return binder.AddConverter(mvc.ConverterFunc[string, starterAdviceTenantID](func(value string) (starterAdviceTenantID, error) {
				return starterAdviceTenantID{value: "starter-advice:" + value}, nil
			}))
		}),
	)
	controller := mvc.NewRestController("starter-advice-search",
		mvc.GET("/advice-binder", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterAdviceSearchPayload, error) {
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
		})),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterConversionConfiguration{},
			mvc.NewConfiguration("test.mvc.advice-init-binder", controller).WithControllerAdvices(advice),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/advice-binder?page=goark&tenant=blue", http.StatusOK)
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
	service, err := goark.Get[*convert.Service](t.Context(), appContext, gbcweb.BeanNameConversionService)
	if err != nil {
		t.Fatalf("resolve conversion service failed: %v", err)
	}
	if _, err := convert.Convert[starterAdviceTenantID](service, "blue"); err == nil {
		t.Fatal("starter conversion service should not see advice converter")
	}
}

type starterConversionConfiguration struct{}

func (starterConversionConfiguration) Name() string {
	return "test.web.conversion"
}

func (starterConversionConfiguration) Order() int {
	return 0
}

func (c starterConversionConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterConversionConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	if err := gbcweb.RegisterConverter(config.Registry(), "testStringIntConverter", convert.ConverterFunc[string, int](func(value string) (int, error) {
		return len(value) + 100, nil
	})); err != nil {
		return err
	}
	return gbcweb.RegisterConverter(config.Registry(), "testStringTenantIDConverter", convert.ConverterFunc[string, starterTenantID](func(value string) (starterTenantID, error) {
		return starterTenantID{value: "tenant:" + value}, nil
	}))
}

func starterBinderPayloadFromInput(input starterBinderInput) starterBinderPayload {
	out := starterBinderPayload{
		Name:     input.Name,
		System:   input.System,
		Admin:    input.Admin,
		Roles:    input.Roles,
		Metadata: input.Metadata,
	}
	if input.Profile != nil {
		out.ProfileEmail = input.Profile.Email
		out.ProfileAdmin = input.Profile.Admin
	}
	return out
}

func starterPreferencePayloadFromInput(input starterPreferenceInput) starterPreferencePayload {
	out := starterPreferencePayload{
		Theme:      input.Theme,
		TagsNil:    input.Tags == nil,
		TagsLength: len(input.Tags),
	}
	if input.Notify != nil {
		out.NotifySet = true
		out.Notify = *input.Notify
	}
	if input.Confirm != nil {
		out.ConfirmSet = true
		out.Confirm = *input.Confirm
	}
	if input.Profile != nil {
		out.ProfileSet = true
		out.Subscribed = input.Profile.Subscribed
	}
	return out
}

func assertStarterPreferencePayload(t *testing.T, payload starterPreferencePayload) {
	t.Helper()
	if payload.Theme != "dark" ||
		!payload.NotifySet ||
		payload.Notify ||
		!payload.ConfirmSet ||
		!payload.Confirm ||
		!payload.ProfileSet ||
		payload.Subscribed ||
		payload.TagsNil ||
		payload.TagsLength != 0 {
		t.Fatalf("payload = %#v, want field prefix binding", payload)
	}
}
