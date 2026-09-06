package gbcweb_test

import (
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenControllerInitBinderExists_shouldUseScopedConvertersThroughArkhos(t *testing.T) {
	localController := mvc.NewRestController("starter-local-search",
		mvc.GET("/init-binder/local", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterScopedSearchPayload, error) {
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
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.AddConverter(mvc.ConverterFunc[string, starterScopedTenantID](func(value string) (starterScopedTenantID, error) {
			return starterScopedTenantID{value: "starter-local:" + value}, nil
		}))
	}))
	plainController := mvc.NewRestController("starter-plain-search",
		mvc.GET("/init-binder/plain", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterScopedSearchPayload, error) {
			criteria, err := mvc.ModelAttribute[starterScopedSearchCriteria](ctx)
			if err != nil {
				return starterScopedSearchPayload{}, err
			}
			return starterScopedSearchPayload{
				Page:   criteria.Page,
				Tenant: criteria.Tenant.value,
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
			mvc.NewConfiguration("test.mvc.init-binder", localController, plainController),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	localSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/init-binder/local?page=goark&tenant=blue", http.StatusOK)
	var localPayload starterScopedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(localSnapshot.body), &localPayload); err != nil {
		t.Fatalf("local init binder json invalid: %v", err)
	}
	if localPayload.Page != 105 ||
		localPayload.Tenant != "starter-local:blue" ||
		localPayload.ParamTenant != "starter-local:blue" {
		t.Fatalf("local init binder payload = %#v, want scoped conversion", localPayload)
	}

	requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/init-binder/plain?page=goark&tenant=blue", http.StatusBadRequest)
}

func TestAutoConfigure_whenInitBinderFieldFiltersExist_shouldBindModelAttributesThroughArkhos(t *testing.T) {
	allowedController := mvc.NewRestController("starter-field-allowed",
		mvc.GET("/field-binder/allowed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterBinderPayload, error) {
			input, err := mvc.ModelAttribute[starterBinderInput](ctx)
			if err != nil {
				return starterBinderPayload{}, err
			}
			return starterBinderPayloadFromInput(input), nil
		})),
	).WithInitBinders(mvc.BinderInitializerFunc(func(_ *arkweb.Context, binder *mvc.DataBinder) error {
		return binder.SetAllowedFields("name", "profile.email", "roles[*].name", "metadata[department]")
	}))
	disallowedController := mvc.NewRestController("starter-field-disallowed",
		mvc.GET("/field-binder/disallowed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterBinderPayload, error) {
			input, err := mvc.ModelAttribute[starterBinderInput](ctx)
			if err != nil {
				return starterBinderPayload{}, err
			}
			return starterBinderPayloadFromInput(input), nil
		})),
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
			mvc.NewConfiguration("test.mvc.field-binders", allowedController, disallowedController).WithControllerAdvices(advice),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	baseURL := starterServerURL(t, app)
	query := "?name=ada&system=root&admin=true&profile.email=ada@example.test&profile.admin=true&" +
		"roles[0].name=reader&roles[0].admin=true&metadata[department]=engineering&metadata[secret]=root"

	allowedSnapshot := requestUntilStatusSnapshot(t, baseURL+"/field-binder/allowed"+query, http.StatusOK)
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
	if len(allowedPayload.Roles) != 1 || allowedPayload.Roles[0].Name != "reader" || allowedPayload.Roles[0].Admin {
		t.Fatalf("allowed roles = %#v, want only role name", allowedPayload.Roles)
	}
	if allowedPayload.Metadata["department"] != "engineering" {
		t.Fatalf("allowed metadata = %#v, want department", allowedPayload.Metadata)
	}
	if _, ok := allowedPayload.Metadata["secret"]; ok {
		t.Fatalf("allowed metadata = %#v, want secret skipped", allowedPayload.Metadata)
	}

	disallowedSnapshot := requestUntilStatusSnapshot(t, baseURL+"/field-binder/disallowed"+query, http.StatusOK)
	var disallowedPayload starterBinderPayload
	if err := arkjson.Unmarshal(nil, []byte(disallowedSnapshot.body), &disallowedPayload); err != nil {
		t.Fatalf("disallowed field binder json invalid: %v", err)
	}
	if disallowedPayload.Name != "ada" ||
		disallowedPayload.System != "" ||
		!disallowedPayload.Admin ||
		disallowedPayload.ProfileEmail != "ada@example.test" ||
		!disallowedPayload.ProfileAdmin {
		t.Fatalf("disallowed payload = %#v, want only globally disallowed system field skipped", disallowedPayload)
	}
	if len(disallowedPayload.Roles) != 1 || disallowedPayload.Roles[0].Name != "reader" || !disallowedPayload.Roles[0].Admin {
		t.Fatalf("disallowed roles = %#v, want role fields except global system", disallowedPayload.Roles)
	}
	if disallowedPayload.Metadata["department"] != "engineering" || disallowedPayload.Metadata["secret"] != "root" {
		t.Fatalf("disallowed metadata = %#v, want non-system metadata preserved", disallowedPayload.Metadata)
	}
}

func TestAutoConfigure_whenFieldPrefixesExist_shouldBindModelAttributesThroughArkhos(t *testing.T) {
	defaultController := mvc.NewRestController("starter-field-prefix-default",
		mvc.GET("/field-prefix/default", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterPreferencePayload, error) {
			input, err := mvc.ModelAttribute[starterPreferenceInput](ctx)
			if err != nil {
				return starterPreferencePayload{}, err
			}
			return starterPreferencePayloadFromInput(input), nil
		})),
	)
	customController := mvc.NewRestController("starter-field-prefix-custom",
		mvc.GET("/field-prefix/custom", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterPreferencePayload, error) {
			input, err := mvc.ModelAttribute[starterPreferenceInput](ctx)
			if err != nil {
				return starterPreferencePayload{}, err
			}
			return starterPreferencePayloadFromInput(input), nil
		})),
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
	defaultSnapshot := requestUntilStatusSnapshot(t, baseURL+"/field-prefix/default?!theme=dark&_notify=on&confirm=true&_confirm=on&_profile.subscribed=on&_tags=on", http.StatusOK)
	var defaultPayload starterPreferencePayload
	if err := arkjson.Unmarshal(nil, []byte(defaultSnapshot.body), &defaultPayload); err != nil {
		t.Fatalf("default field prefix json invalid: %v", err)
	}
	assertStarterPreferencePayload(t, defaultPayload)

	customSnapshot := requestUntilStatusSnapshot(t, baseURL+"/field-prefix/custom?~theme=dark&__notify=on&confirm=true&__confirm=on&__profile.subscribed=on&__tags=on", http.StatusOK)
	var customPayload starterPreferencePayload
	if err := arkjson.Unmarshal(nil, []byte(customSnapshot.body), &customPayload); err != nil {
		t.Fatalf("custom field prefix json invalid: %v", err)
	}
	assertStarterPreferencePayload(t, customPayload)
}

func TestAutoConfigure_whenEmptyArrayIndexFieldsExist_shouldBindModelAttributesThroughArkhos(t *testing.T) {
	controller := mvc.NewRestController("starter-empty-array-index",
		mvc.GET("/field-array-indices", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterArrayPayload, error) {
			input, err := mvc.ModelAttribute[starterArrayInput](ctx)
			if err != nil {
				return starterArrayPayload{}, err
			}
			out := starterArrayPayload{Tags: input.Tags}
			if input.Profile != nil {
				out.Aliases = input.Profile.Aliases
			}
			return out, nil
		})),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.empty-array-index", controller),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/field-array-indices?tags[]=red&tags[]=blue&profile.aliases[]=core&profile.aliases[]=web", http.StatusOK)
	var payload starterArrayPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("empty array index json invalid: %v", err)
	}
	if len(payload.Tags) != 2 ||
		payload.Tags[0] != "red" ||
		payload.Tags[1] != "blue" ||
		len(payload.Aliases) != 2 ||
		payload.Aliases[0] != "core" ||
		payload.Aliases[1] != "web" {
		t.Fatalf("payload = %#v, want empty array index fields", payload)
	}
}

func TestAutoConfigure_whenSuppressedFieldsExist_shouldExposeBindingResultThroughArkhos(t *testing.T) {
	controller := mvc.NewRestController("starter-suppressed-fields",
		mvc.GET("/field-binder/suppressed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterSuppressedPayload, error) {
			input, result, err := mvc.ModelAttributeResult[starterBinderInput](ctx)
			if err != nil {
				return starterSuppressedPayload{}, err
			}
			return starterSuppressedPayload{
				Name:             input.Name,
				Admin:            input.Admin,
				SuppressedFields: result.SuppressedFields(),
			}, nil
		})),
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

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/field-binder/suppressed?name=ada&admin=true&profile.email=ada@example.test", http.StatusOK)
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
