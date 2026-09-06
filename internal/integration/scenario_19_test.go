package gbcweb_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	webclient "goark.dev/goark/web/client"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenHTTPClientBuilderCustomizersExist_shouldApplyInOrder(t *testing.T) {
	serverErrors := make(chan error, 1)
	api := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/customized" {
				failHTTPServer(
					serverErrors,
					writer,
					"path = %q, want /customized",
					request.URL.Path,
				)
				return
			}
			values := request.Header.Values("X-Chain")
			if len(values) != 2 || values[0] != "first" || values[1] != "second" {
				failHTTPServer(
					serverErrors,
					writer,
					"X-Chain = %#v, want first then second",
					values,
				)
				return
			}
			phase, err := request.Cookie("phase")
			if err != nil || phase.Value != "first" {
				failHTTPServer(serverErrors, writer, "phase cookie = %#v err %v", phase, err)
				return
			}
			mode, err := request.Cookie("mode")
			if err != nil || mode.Value != "second" {
				failHTTPServer(serverErrors, writer, "mode cookie = %#v err %v", mode, err)
				return
			}
			_, _ = io.WriteString(writer, "customized")
		}),
	)
	defer api.Close()

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
			gbcweb.WithHTTPClientBaseURL(api.URL),
		)),
		boot.WithConfiguration(starterHTTPClientCustomizerConfiguration{}),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	defaultClient, err := goark.Get[*webclient.Client](
		t.Context(),
		appContext,
		gbcweb.BeanNameHTTPClient,
	)
	if err != nil {
		t.Fatalf("resolve http client failed: %v", err)
	}
	response, err := defaultClient.Get(t.Context(), "/customized")
	if err != nil {
		t.Fatalf("customized client get failed: %v", err)
	}
	if response.BodyString() != "customized" {
		t.Fatalf("body = %q, want customized", response.BodyString())
	}
	assertNoHTTPServerError(t, serverErrors)
}

func TestAutoConfigure_whenModelAttributeBindingResultExists_shouldServeThroughArkhos(
	t *testing.T,
) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.model-attribute-binding-result", mvc.NewRestController(
				"bindingSearch",
				mvc.GET(
					"/binding-result/search",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterModelAttributeBindingPayload, error) {
							input, result, err := mvc.ModelAttributeResult[starterBindingSearchCriteria](
								ctx,
							)
							if err != nil {
								return starterModelAttributeBindingPayload{}, err
							}
							field, _ := result.FieldError("name")
							return starterModelAttributeBindingPayload{
								Valid:        result.Valid(),
								BindingError: result.BindingError() != nil,
								Field:        field.Path(),
								Name:         input.Name,
								Page:         input.Page,
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

	serverURL := starterServerURL(t, app)
	bindingSnapshot := requestUntilStatusSnapshot(
		t,
		serverURL+"/binding-result/search?name=goark&page=bad",
		http.StatusOK,
	)
	var bindingPayload starterModelAttributeBindingPayload
	if err := arkjson.Unmarshal(nil, []byte(bindingSnapshot.body), &bindingPayload); err != nil {
		t.Fatalf("binding result json invalid: %v", err)
	}
	if bindingPayload.Valid || !bindingPayload.BindingError || bindingPayload.Field != "" ||
		bindingPayload.Name != "goark" ||
		bindingPayload.Page != 0 {
		t.Fatalf("binding payload = %#v, want captured binding error", bindingPayload)
	}

	validationSnapshot := requestUntilStatusSnapshot(
		t,
		serverURL+"/binding-result/search?page=2",
		http.StatusOK,
	)
	var validationPayload starterModelAttributeBindingPayload
	if err := arkjson.Unmarshal(nil, []byte(validationSnapshot.body), &validationPayload); err != nil {
		t.Fatalf("validation result json invalid: %v", err)
	}
	if validationPayload.Valid || validationPayload.BindingError ||
		validationPayload.Field != "name" ||
		validationPayload.Page != 2 {
		t.Fatalf("validation payload = %#v, want captured validation error", validationPayload)
	}
}

func TestAutoConfigure_whenEmptyArrayRequestParamsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.empty-array-request-param", mvc.NewRestController(
				"requestParams",
				mvc.GET(
					"/search",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterRequestParamPayload, error) {
							query, err := mvc.RequestParamString(ctx, "q")
							if err != nil {
								return starterRequestParamPayload{}, err
							}
							tags, err := mvc.RequestParamStrings(ctx, "tag")
							if err != nil {
								return starterRequestParamPayload{}, err
							}
							ids, err := mvc.RequestParamInt64s(ctx, "id")
							if err != nil {
								return starterRequestParamPayload{}, err
							}
							return starterRequestParamPayload{
								Query: query,
								Tags:  tags,
								IDs:   ids,
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
		starterServerURL(t, app)+"/search?q[]=goark&tag[]=web&tag[]=mvc&id[]=1&id[]=2",
		http.StatusOK,
	)
	var got starterRequestParamPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got.Query != "goark" ||
		!reflect.DeepEqual(got.Tags, []string{"web", "mvc"}) ||
		!reflect.DeepEqual(got.IDs, []int64{1, 2}) {
		t.Fatalf("payload = %#v, want empty array index request params", got)
	}
}

func TestAutoConfigure_whenLocaleHelpersExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.locale", mvc.NewRestController("locale",
				mvc.GET("/locale", mvc.Entity(starterLocaleEntity)),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/locale",
			nil,
		)
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
	if !payload.OK || payload.Locale != "zh-CN" || payload.Language != "zh" ||
		payload.Region != "CN" ||
		payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want locale details", payload)
	}
}

func (starterHTTPClientCustomizerConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	if err := gbcweb.RegisterHTTPClientBuilderCustomizer(
		config.Registry(), "testFirstHTTPClientCustomizer",
		gbcweb.HTTPClientBuilderCustomizerFunc(
			func(
				ctx context.Context,
				builder *webclient.Builder,
			) (*webclient.Builder, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return builder.DefaultHeader("X-Chain", "first").DefaultCookieValue("phase", "first"), nil
			}), container.WithOrder(-100)); err != nil {
		return err
	}
	return gbcweb.RegisterHTTPClientBuilderCustomizer(
		config.Registry(),
		"testSecondHTTPClientCustomizer",
		gbcweb.HTTPClientBuilderCustomizerFunc(
			func(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return builder.DefaultHeader("X-Chain", "second").
						DefaultCookieValue("mode", "second"),
					nil
			},
		),
		container.WithOrder(100),
	)
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

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q failed: %v", path, err)
	}
}

type starterSuppressedPayload struct {
	Name             string   `json:"name"`
	Admin            bool     `json:"admin"`
	SuppressedFields []string `json:"suppressedFields"`
}

type starterArrayInput struct {
	Tags    []string             `form:"tags"    json:"tags"`
	Profile *starterArrayProfile `form:"profile" json:"profile"`
}

func (starterHTTPClientCustomizerConfiguration) Name() string {
	return "test.web.http-client-customizer"
}

func (starterRequestBodyAdviceConfiguration) Name() string {
	return "test.web.request-body-advice"
}

type starterEventPayload struct {
	State string `json:"state"`
}
