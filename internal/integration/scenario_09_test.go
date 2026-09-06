package gbcweb_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenRepeatedMatrixVariablesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.matrix-variable-repeated", mvc.NewRestController(
				"matrixValues",
				mvc.GET(
					"/cars/{carId}/owners/{ownerId}",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterMatrixVariableValuesPayload, error) {
							colors, err := mvc.MatrixVariableStrings(ctx, "color")
							if err != nil {
								return starterMatrixVariableValuesPayload{}, err
							}
							codes, err := mvc.MatrixVariableInts(ctx, "code")
							if err != nil {
								return starterMatrixVariableValuesPayload{}, err
							}
							ownerColors, err := mvc.MatrixVariableStrings(
								ctx,
								"color",
								mvc.WithMatrixPathVariable("ownerId"),
							)
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
							ownerMatrix, err := mvc.MatrixVariableMap(
								ctx,
								mvc.WithMatrixPathVariable("ownerId"),
							)
							if err != nil {
								return starterMatrixVariableValuesPayload{}, err
							}
							ownerValues, err := mvc.MatrixVariableValuesMap(
								ctx,
								mvc.WithMatrixPathVariable("ownerId"),
							)
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
		starterServerURL(
			t,
			app,
		)+"/cars/42;color=red;color=blue;code=1,2;code=3/owners/7;color=black;color=white;q=owner",
		http.StatusOK,
	)
	var got starterMatrixVariableValuesPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Colors, []string{"red", "blue", "black", "white"}) ||
		!reflect.DeepEqual(got.Codes, []int{1, 2, 3}) ||
		!reflect.DeepEqual(got.OwnerColors, []string{"black", "white"}) {
		t.Fatalf("payload = %#v, want repeated matrix variable slices", got)
	}
	if !reflect.DeepEqual(
		got.Matrix,
		map[string]string{"color": "red", "code": "1,2", "q": "owner"},
	) {
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

func TestAutoConfigure_whenTypedAttributesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterAttributeConfiguration{},
			mvc.NewConfiguration(
				"test.mvc.typed-attributes",
				mvc.NewRestController(
					"typedAttributes",
					mvc.GET(
						"/attributes/typed",
						mvc.JSON(
							http.StatusOK,
							func(ctx *arkweb.Context) (starterAttributePayload, error) {
								profile, err := mvc.RequestAttribute[starterAttributeProfile](
									ctx,
									"profile",
								)
								if err != nil {
									return starterAttributePayload{}, err
								}
								limit, err := mvc.RequestAttribute[int](ctx, "limit")
								if err != nil {
									return starterAttributePayload{}, err
								}
								traceID, err := mvc.SessionAttribute[string](ctx, "traceID")
								if err != nil {
									return starterAttributePayload{}, err
								}
								return starterAttributePayload{
									TraceID:   traceID,
									ProfileID: profile.ID,
									Limit:     limit,
								}, nil
							},
						),
					),
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
		starterServerURL(t, app)+"/attributes/typed",
		http.StatusOK,
	)
	var payload starterAttributePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if payload.TraceID != "trace-1" || payload.ProfileID != "p-1" || payload.Limit != 42 {
		t.Fatalf("payload = %#v, want typed attributes", payload)
	}
}

func TestAutoConfigure_whenMVCControllerExists_shouldServeRequestWithArkhos(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc", mvc.NewController(
			"health",
			mvc.GET(
				"/healthz",
				mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return map[string]string{"status": "UP"}, nil
				}),
			),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](
		t.Context(),
		appContext,
		gbcarkhos.BeanNameServer,
	)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	host, _, err := net.SplitHostPort(server.Address())
	if err != nil {
		t.Fatalf("split embedded server address failed: %v", err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("server host = %q, want configured host", host)
	}
	body := requestUntilOK(t, server.URL()+"/healthz")
	if body != `{"status":"UP"}` {
		t.Fatalf("body = %q, want health json", body)
	}
}

func requestUntilStatusSnapshotWithMethod(
	t *testing.T,
	method string,
	target string,
	statusCode int,
) responseSnapshot {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(3 * time.Second)
	for {
		request, requestErr := http.NewRequestWithContext(t.Context(), method, target, nil)
		if requestErr != nil {
			t.Fatalf("%s %s request invalid: %v", method, target, requestErr)
		}
		response, err := client.Do(request)
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read/close response = %v/%v", readErr, closeErr)
			}
			if response.StatusCode == statusCode {
				return responseSnapshot{
					body:   string(body),
					header: response.Header.Clone(),
				}
			}
			t.Fatalf(
				"status = %d, want %d, body = %q",
				response.StatusCode,
				statusCode,
				string(body),
			)
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s %s did not succeed before deadline: %v", method, target, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
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

func assertNoHTTPServerError(t *testing.T, errors <-chan error) {
	t.Helper()
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}

func (c starterLocaleChangeConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterAttributePayload struct {
	TraceID   string `json:"traceId"`
	ProfileID string `json:"profileId"`
	Limit     int    `json:"limit"`
}

func (starterTokenConverter) CanWrite(value any, mediaType string) bool {
	_, ok := value.(starterTokenOutput)
	return ok && strings.HasPrefix(mediaType, starterTokenMediaType)
}

type starterScopedTenantID struct {
	value string
}

func (starterAttributeConfiguration) Name() string {
	return "test.web.typed-attributes"
}

type starterErrorMapperDependency struct{}

type starterRequestBodyAdviceConfiguration struct{}
