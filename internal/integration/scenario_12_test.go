package gbcweb_test

import (
	"errors"
	"io"
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
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenNamedRequestPartsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.named-request-parts", mvc.NewRestController(
				"namedParts",
				mvc.POST(
					"/parts/files",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterRequestPartsPayload, error) {
							parts, err := mvc.RequestPartsByName(ctx, "file")
							if err != nil {
								return starterRequestPartsPayload{}, err
							}
							optional, err := mvc.RequestPartsByName(
								ctx,
								"missing",
								mvc.WithRequired(false),
							)
							if err != nil {
								return starterRequestPartsPayload{}, err
							}
							payload := starterRequestPartsPayload{
								Names:         make([]string, 0, len(parts)),
								Bodies:        make([]string, 0, len(parts)),
								OptionalCount: len(optional),
							}
							for _, part := range parts {
								reader, err := part.Open()
								if err != nil {
									return starterRequestPartsPayload{}, err
								}
								body, readErr := io.ReadAll(reader)
								closeErr := reader.Close()
								if readErr != nil {
									return starterRequestPartsPayload{}, readErr
								}
								if closeErr != nil {
									return starterRequestPartsPayload{}, closeErr
								}
								payload.Names = append(payload.Names, part.SubmittedFileName())
								payload.Bodies = append(payload.Bodies, string(body))
							}
							return payload, nil
						},
					),
					mvc.WithConsumes("multipart/form-data"),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterNamedRequestPartsBody(t)
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/parts/files",
			strings.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusOK)

	var payload starterRequestPartsPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if len(payload.Names) != 2 ||
		payload.Names[0] != "first.txt" ||
		payload.Names[1] != "second.txt" ||
		len(payload.Bodies) != 2 ||
		payload.Bodies[0] != "first" ||
		payload.Bodies[1] != "second" ||
		payload.OptionalCount != 0 {
		t.Fatalf("payload = %#v, want named request parts", payload)
	}
}

func TestAutoConfigure_whenOptionalRequestBodyExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.optional-request-body", mvc.NewRestController(
				"optionalBody",
				mvc.POST(
					"/body/optional",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterOptionalBodyPayload, error) {
							input, present, err := mvc.OptionalValidatedRequestBody[starterOptionalBodyInput](
								ctx,
							)
							if err != nil {
								return starterOptionalBodyPayload{}, err
							}
							return starterOptionalBodyPayload{
								Present: present,
								Name:    input.Name,
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

	baseURL := starterServerURL(t, app)
	absent := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			baseURL+"/body/optional",
			nil,
		)
	}, http.StatusOK)
	if absent.body != `{"present":false,"name":""}` {
		t.Fatalf("absent body = %q, want optional body skipped", absent.body)
	}

	present := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			baseURL+"/body/optional",
			strings.NewReader(`{"name":"goark"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)
	if present.body != `{"present":true,"name":"goark"}` {
		t.Fatalf("present body = %q, want optional body bound", present.body)
	}
}

func TestAutoConfigure_whenMultipartBindingResultExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration(
				"test.mvc.multipart-binding-result",
				mvc.NewRestController("bindingUploads",
					mvc.POST("/binding-result/uploads", mvc.BindMultipartResult(
						http.StatusOK,
						func(
							_ *arkweb.Context,
							input starterBindingUploadRequest,
							result mvc.BindingResult,
						) (starterMultipartBindingPayload, error) {
							field, _ := result.FieldError("title")
							return starterMultipartBindingPayload{
								Valid:    result.Valid(),
								Field:    field.Path(),
								Filename: input.File.SubmittedFileName(),
							}, nil
						},
					)),
				),
			),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterFileOnlyMultipartBody(t)
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/binding-result/uploads",
			strings.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)

	var payload starterMultipartBindingPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("multipart binding json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "title" || payload.Filename != "profile.txt" {
		t.Fatalf("multipart binding payload = %#v, want captured validation error", payload)
	}
}

func TestAutoConfigure_whenHandlerReturnsError_shouldUseProblemDetails(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.problem", mvc.NewController(
			"problem",
			mvc.GET(
				"/problem",
				mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return nil, errors.New("internal secret")
				}),
			),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/problem",
		http.StatusInternalServerError,
	)
	if snapshot.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("content type = %q, want problem json", snapshot.header.Get("Content-Type"))
	}
	if !strings.Contains(snapshot.body, `"status":500`) ||
		!strings.Contains(snapshot.body, `"instance":"/problem"`) ||
		strings.Contains(snapshot.body, "internal secret") {
		t.Fatalf("problem body = %q", snapshot.body)
	}
}

func assertConfigurationCount(
	t *testing.T,
	descriptors []goark.ConfigurationDescriptor,
	name string,
	want int,
) {
	t.Helper()

	got := 0
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			got++
		}
	}
	if got != want {
		t.Fatalf("configuration %q count = %d, want %d", name, got, want)
	}
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

type starterPreferenceInput struct {
	Theme   string                    `form:"theme"   json:"theme"`
	Notify  *bool                     `form:"notify"  json:"notify"`
	Confirm *bool                     `form:"confirm" json:"confirm"`
	Profile *starterPreferenceProfile `form:"profile" json:"profile"`
	Tags    []string                  `form:"tags"    json:"tags"`
}

type starterBindingResultPayload struct {
	Valid bool   `json:"valid"`
	Path  string `json:"path"`
	Code  string `json:"code"`
	Name  string `json:"name"`
}

type starterAdviceSearchCriteria struct {
	Page   int                   `form:"page"`
	Tenant starterAdviceTenantID `form:"tenant"`
}

func requestUntilOK(t *testing.T, target string) string {
	return requestUntilStatus(t, target, http.StatusOK)
}

func (starterLocaleChangeConfiguration) Order() int {
	return 0
}

type starterValidatorRequest struct {
	Name string `json:"name"`
}

type starterAttributeConfiguration struct{}
