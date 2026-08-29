package gbcweb_test

import (
	"context"
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	"goark.dev/goark/web/mvc"
)

type starterTenantID struct {
	value string
}

type starterScopedTenantID struct {
	value string
}

type starterAdviceTenantID struct {
	value string
}

type starterConversionPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
}

type starterSearchCriteria struct {
	Page   int             `form:"page"`
	Tenant starterTenantID `form:"tenant"`
}

type starterScopedSearchCriteria struct {
	Page   int                   `form:"page"`
	Tenant starterScopedTenantID `form:"tenant"`
}

type starterAdviceSearchCriteria struct {
	Page   int                   `form:"page"`
	Tenant starterAdviceTenantID `form:"tenant"`
}

type starterSearchOwner struct {
	Name  string `form:"name"`
	Level int    `form:"level"`
}

type starterNestedSearchCriteria struct {
	Owner *starterSearchOwner `form:"owner"`
	Page  int                 `form:"page"`
}

type starterIndexedSearchCriteria struct {
	Owners []starterIndexedSearchOwner `form:"owners"`
	Page   int                         `form:"page"`
}

type starterIndexedSearchOwner struct {
	Name    string   `form:"name"`
	Level   int      `form:"level"`
	Aliases []string `form:"aliases"`
}

type starterMappedSearchCriteria struct {
	Filters map[string]int      `form:"filters"`
	Tags    map[string][]string `form:"tags"`
}

type starterBinderProfile struct {
	Email string `form:"email" json:"email"`
	Admin bool   `form:"admin" json:"admin"`
}

type starterBinderRole struct {
	Name  string `form:"name" json:"name"`
	Admin bool   `form:"admin" json:"admin"`
}

type starterBinderInput struct {
	Name     string                `form:"name" json:"name"`
	System   string                `form:"system" json:"system"`
	Admin    bool                  `form:"admin" json:"admin"`
	Profile  *starterBinderProfile `form:"profile" json:"profile"`
	Roles    []starterBinderRole   `form:"roles" json:"roles"`
	Metadata map[string]string     `form:"metadata" json:"metadata"`
}

type starterPreferenceProfile struct {
	Subscribed bool `form:"subscribed" json:"subscribed"`
}

type starterPreferenceInput struct {
	Theme   string                    `form:"theme" json:"theme"`
	Notify  *bool                     `form:"notify" json:"notify"`
	Confirm *bool                     `form:"confirm" json:"confirm"`
	Profile *starterPreferenceProfile `form:"profile" json:"profile"`
	Tags    []string                  `form:"tags" json:"tags"`
}

type starterArrayProfile struct {
	Aliases []string `form:"aliases" json:"aliases"`
}

type starterArrayInput struct {
	Tags    []string             `form:"tags" json:"tags"`
	Profile *starterArrayProfile `form:"profile" json:"profile"`
}

type starterSearchPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
}

type starterScopedSearchPayload struct {
	Page        int    `json:"page"`
	Tenant      string `json:"tenant"`
	ParamTenant string `json:"paramTenant"`
}

type starterAdviceSearchPayload struct {
	Page        int    `json:"page"`
	Tenant      string `json:"tenant"`
	ParamTenant string `json:"paramTenant"`
}

type starterNestedSearchPayload struct {
	OwnerName  string `json:"ownerName"`
	OwnerLevel int    `json:"ownerLevel"`
	Page       int    `json:"page"`
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

type starterMappedSearchPayload struct {
	Level  int      `json:"level"`
	Roles  []string `json:"roles"`
	Groups []string `json:"groups"`
}

type starterBinderPayload struct {
	Name         string              `json:"name"`
	System       string              `json:"system"`
	Admin        bool                `json:"admin"`
	ProfileEmail string              `json:"profileEmail"`
	ProfileAdmin bool                `json:"profileAdmin"`
	Roles        []starterBinderRole `json:"roles"`
	Metadata     map[string]string   `json:"metadata"`
}

type starterSuppressedPayload struct {
	Name             string   `json:"name"`
	Admin            bool     `json:"admin"`
	SuppressedFields []string `json:"suppressedFields"`
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

type starterArrayPayload struct {
	Tags    []string `json:"tags"`
	Aliases []string `json:"aliases"`
}

func TestAutoConfigure_whenConvertersExist_shouldBindMVCRequestParameters(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterConversionConfiguration{},
			mvc.NewConfiguration("test.mvc.conversion", mvc.NewController("conversion",
				mvc.GET("/conversion", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterConversionPayload, error) {
					page, err := mvc.RequestParamInt(ctx, "page")
					if err != nil {
						return starterConversionPayload{}, err
					}
					tenant, err := mvc.RequestParamAs[starterTenantID](ctx, "tenant")
					if err != nil {
						return starterConversionPayload{}, err
					}
					return starterConversionPayload{
						Page:   page,
						Tenant: tenant.value,
					}, nil
				})),
				mvc.GET("/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterSearchPayload, error) {
					criteria, err := mvc.ModelAttribute[starterSearchCriteria](ctx)
					if err != nil {
						return starterSearchPayload{}, err
					}
					return starterSearchPayload{
						Page:   criteria.Page,
						Tenant: criteria.Tenant.value,
					}, nil
				})),
				mvc.GET("/search/nested", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterNestedSearchPayload, error) {
					criteria, err := mvc.ModelAttribute[starterNestedSearchCriteria](ctx)
					if err != nil {
						return starterNestedSearchPayload{}, err
					}
					return starterNestedSearchPayload{
						OwnerName:  criteria.Owner.Name,
						OwnerLevel: criteria.Owner.Level,
						Page:       criteria.Page,
					}, nil
				})),
				mvc.GET("/search/indexed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterIndexedSearchPayload, error) {
					criteria, err := mvc.ModelAttribute[starterIndexedSearchCriteria](ctx)
					if err != nil {
						return starterIndexedSearchPayload{}, err
					}
					return starterIndexedSearchPayload{
						FirstName:   criteria.Owners[0].Name,
						FirstLevel:  criteria.Owners[0].Level,
						FirstAlias:  criteria.Owners[0].Aliases[0],
						SecondName:  criteria.Owners[1].Name,
						SecondLevel: criteria.Owners[1].Level,
						SecondAlias: criteria.Owners[1].Aliases[0],
						Page:        criteria.Page,
					}, nil
				})),
				mvc.GET("/search/mapped", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterMappedSearchPayload, error) {
					criteria, err := mvc.ModelAttribute[starterMappedSearchCriteria](ctx)
					if err != nil {
						return starterMappedSearchPayload{}, err
					}
					return starterMappedSearchPayload{
						Level:  criteria.Filters["level"],
						Roles:  criteria.Tags["roles"],
						Groups: criteria.Tags["groups"],
					}, nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/conversion?page=abcd&tenant=blue", http.StatusOK)
	var payload starterConversionPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("conversion json invalid: %v", err)
	}
	if payload.Page != 104 || payload.Tenant != "tenant:blue" {
		t.Fatalf("payload = %#v, want converted parameters", payload)
	}

	searchSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search?page=abcde&tenant=green", http.StatusOK)
	var searchPayload starterSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(searchSnapshot.body), &searchPayload); err != nil {
		t.Fatalf("search json invalid: %v", err)
	}
	if searchPayload.Page != 105 || searchPayload.Tenant != "tenant:green" {
		t.Fatalf("search payload = %#v, want converted model attribute", searchPayload)
	}

	nestedSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search/nested?owner.name=ada&owner.level=admin&page=xy", http.StatusOK)
	var nestedPayload starterNestedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(nestedSnapshot.body), &nestedPayload); err != nil {
		t.Fatalf("nested search json invalid: %v", err)
	}
	if nestedPayload.OwnerName != "ada" || nestedPayload.OwnerLevel != 105 || nestedPayload.Page != 102 {
		t.Fatalf("nested search payload = %#v, want nested model attribute", nestedPayload)
	}

	indexedSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search/indexed?"+
		"owners[0].name=ada&owners[0].level=admin&owners[0].aliases[0]=lead&"+
		"owners[1].name=linus&owners[1].level=kernel&owners[1].aliases[0]=maintainer&page=xy", http.StatusOK)
	var indexedPayload starterIndexedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(indexedSnapshot.body), &indexedPayload); err != nil {
		t.Fatalf("indexed search json invalid: %v", err)
	}
	if indexedPayload.FirstName != "ada" ||
		indexedPayload.FirstLevel != 105 ||
		indexedPayload.FirstAlias != "lead" ||
		indexedPayload.SecondName != "linus" ||
		indexedPayload.SecondLevel != 106 ||
		indexedPayload.SecondAlias != "maintainer" ||
		indexedPayload.Page != 102 {
		t.Fatalf("indexed search payload = %#v, want indexed model attribute", indexedPayload)
	}

	mappedSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search/mapped?"+
		"filters[level]=admin&tags[roles]=admin,ops&tags[groups]=core&tags[groups]=web", http.StatusOK)
	var mappedPayload starterMappedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(mappedSnapshot.body), &mappedPayload); err != nil {
		t.Fatalf("mapped search json invalid: %v", err)
	}
	if mappedPayload.Level != 105 ||
		len(mappedPayload.Roles) != 2 ||
		mappedPayload.Roles[0] != "admin" ||
		mappedPayload.Roles[1] != "ops" ||
		len(mappedPayload.Groups) != 2 ||
		mappedPayload.Groups[0] != "core" ||
		mappedPayload.Groups[1] != "web" {
		t.Fatalf("mapped search payload = %#v, want mapped model attribute", mappedPayload)
	}
}

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
