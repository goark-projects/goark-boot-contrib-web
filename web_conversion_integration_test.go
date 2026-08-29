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

type starterConversionPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
}

type starterSearchCriteria struct {
	Page   int             `form:"page"`
	Tenant starterTenantID `form:"tenant"`
}

type starterSearchOwner struct {
	Name  string `form:"name"`
	Level int    `form:"level"`
}

type starterNestedSearchCriteria struct {
	Owner *starterSearchOwner `form:"owner"`
	Page  int                 `form:"page"`
}

type starterSearchPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
}

type starterNestedSearchPayload struct {
	OwnerName  string `json:"ownerName"`
	OwnerLevel int    `json:"ownerLevel"`
	Page       int    `json:"page"`
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
