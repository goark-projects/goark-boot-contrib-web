package gbcweb_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	weblocale "goark.dev/goark/web/locale"
	"goark.dev/goark/web/mvc"
)

type starterLocalePayload struct {
	OK         bool   `json:"ok"`
	Locale     string `json:"locale"`
	Language   string `json:"language"`
	Region     string `json:"region"`
	LocaleSize int    `json:"localeSize"`
}

func TestAutoConfigure_whenLocaleHelpersExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.locale", mvc.NewRestController("locale",
			mvc.GET("/locale", mvc.Entity(starterLocaleEntity)),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/locale", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		request.Header.Set("Accept-Language", "en-US;q=0.8, zh-CN;q=0.9")
		return request, nil
	}, http.StatusOK)

	if snapshot.header.Get("Content-Language") != "zh-CN" {
		t.Fatalf("Content-Language = %q, want zh-CN", snapshot.header.Get("Content-Language"))
	}
	var payload starterLocalePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !payload.OK || payload.Locale != "zh-CN" || payload.Language != "zh" || payload.Region != "CN" || payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want locale details", payload)
	}
}

func TestAutoConfigure_whenLocaleChangeInterceptorExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterLocaleChangeConfiguration{},
			mvc.NewConfiguration("test.mvc.locale.change", mvc.NewRestController("localeChange",
				mvc.GET("/locale/change", mvc.Entity(starterLocaleEntity)),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/locale/change?lang=ko-KR", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		request.Header.Set("Accept-Language", "en-US")
		return request, nil
	}, http.StatusOK)

	if snapshot.header.Get("Content-Language") != "ko-KR" {
		t.Fatalf("Content-Language = %q, want ko-KR", snapshot.header.Get("Content-Language"))
	}
	var payload starterLocalePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !payload.OK || payload.Locale != "ko-KR" || payload.Language != "ko" || payload.Region != "KR" || payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want locale changed by interceptor", payload)
	}
}

type starterLocaleChangeConfiguration struct{}

func (starterLocaleChangeConfiguration) Name() string {
	return "test.web.locale.change"
}

func (starterLocaleChangeConfiguration) Order() int {
	return 0
}

func (c starterLocaleChangeConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterLocaleChangeConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	interceptor, err := weblocale.ChangeInterceptor(weblocale.WithParameterName("lang"))
	if err != nil {
		return err
	}
	return goweb.RegisterInterceptor(config.Registry(), "testLocaleChangeInterceptor", interceptor)
}

func starterLocaleEntity(ctx *arkweb.Context) (goweb.ResponseEntity[starterLocalePayload], error) {
	locale, ok := goweb.RequestLocale(ctx)
	locales := goweb.RequestLocales(ctx)
	return goweb.OK(starterLocalePayload{
		OK:         ok,
		Locale:     locale.Tag(),
		Language:   locale.Language(),
		Region:     locale.Region(),
		LocaleSize: len(locales),
	}).WithContentLanguage(locale), nil
}

type starterMatrixVariablePayload struct {
	Owner string `json:"owner"`
	Pet   string `json:"pet"`
	Color string `json:"color"`
}

type starterMatrixVariableValuesPayload struct {
	Colors      []string            `json:"colors"`
	Codes       []int               `json:"codes"`
	OwnerColors []string            `json:"ownerColors"`
	Matrix      map[string]string   `json:"matrix"`
	Values      map[string][]string `json:"values"`
	OwnerMatrix map[string]string   `json:"ownerMatrix"`
	OwnerValues map[string][]string `json:"ownerValues"`
}

func TestAutoConfigure_whenMatrixVariablePathScopeExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.matrix-variable", mvc.NewRestController("matrix",
			mvc.GET("/owners/{ownerId}/pets/{petId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterMatrixVariablePayload, error) {
				owner, err := mvc.MatrixVariableString(ctx, "q", mvc.WithMatrixPathVariable("ownerId"))
				if err != nil {
					return starterMatrixVariablePayload{}, err
				}
				pet, err := mvc.MatrixVariableString(ctx, "q", mvc.WithMatrixPathVariable("petId"))
				if err != nil {
					return starterMatrixVariablePayload{}, err
				}
				color, err := mvc.MatrixVariableString(ctx, "color", mvc.WithMatrixPathVariable("petId"))
				if err != nil {
					return starterMatrixVariablePayload{}, err
				}
				return starterMatrixVariablePayload{Owner: owner, Pet: pet, Color: color}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/owners/42;q=owner/pets/21;q=pet;color=black", http.StatusOK)
	var got starterMatrixVariablePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got != (starterMatrixVariablePayload{Owner: "owner", Pet: "pet", Color: "black"}) {
		t.Fatalf("payload = %#v, want path-scoped matrix variables", got)
	}
}

func TestAutoConfigure_whenRepeatedMatrixVariablesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.matrix-variable-repeated", mvc.NewRestController("matrixValues",
			mvc.GET("/cars/{carId}/owners/{ownerId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterMatrixVariableValuesPayload, error) {
				colors, err := mvc.MatrixVariableStrings(ctx, "color")
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				codes, err := mvc.MatrixVariableInts(ctx, "code")
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				ownerColors, err := mvc.MatrixVariableStrings(ctx, "color", mvc.WithMatrixPathVariable("ownerId"))
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				matrix, err := mvc.MatrixVariableMap(ctx)
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				values, err := mvc.MatrixVariableValuesMap(ctx)
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				ownerMatrix, err := mvc.MatrixVariableMap(ctx, mvc.WithMatrixPathVariable("ownerId"))
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				ownerValues, err := mvc.MatrixVariableValuesMap(ctx, mvc.WithMatrixPathVariable("ownerId"))
				if err != nil {
					return starterMatrixVariableValuesPayload{}, err
				}
				return starterMatrixVariableValuesPayload{
					Colors:      colors,
					Codes:       codes,
					OwnerColors: ownerColors,
					Matrix:      matrix,
					Values:      values,
					OwnerMatrix: ownerMatrix,
					OwnerValues: ownerValues,
				}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/cars/42;color=red;color=blue;code=1,2;code=3/owners/7;color=black;color=white;q=owner", http.StatusOK)
	var got starterMatrixVariableValuesPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Colors, []string{"red", "blue", "black", "white"}) ||
		!reflect.DeepEqual(got.Codes, []int{1, 2, 3}) ||
		!reflect.DeepEqual(got.OwnerColors, []string{"black", "white"}) {
		t.Fatalf("payload = %#v, want repeated matrix variable slices", got)
	}
	if !reflect.DeepEqual(got.Matrix, map[string]string{"color": "red", "code": "1,2", "q": "owner"}) {
		t.Fatalf("matrix = %#v, want first values", got.Matrix)
	}
	if !reflect.DeepEqual(got.Values["color"], []string{"red", "blue", "black", "white"}) ||
		!reflect.DeepEqual(got.Values["code"], []string{"1,2", "3"}) ||
		!reflect.DeepEqual(got.Values["q"], []string{"owner"}) {
		t.Fatalf("values = %#v, want all matrix values", got.Values)
	}
	if !reflect.DeepEqual(got.OwnerMatrix, map[string]string{"color": "black", "q": "owner"}) {
		t.Fatalf("owner matrix = %#v, want first owner values", got.OwnerMatrix)
	}
	if !reflect.DeepEqual(got.OwnerValues["color"], []string{"black", "white"}) ||
		!reflect.DeepEqual(got.OwnerValues["q"], []string{"owner"}) {
		t.Fatalf("owner values = %#v, want all owner matrix values", got.OwnerValues)
	}
}
