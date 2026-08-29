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

type starterSearchPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
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
