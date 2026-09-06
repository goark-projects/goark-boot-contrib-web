package gbcweb_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/problem"
)

func TestAutoConfigure_whenJSONRequestPartExists_shouldBindValidateAndServeThroughArkhos(
	t *testing.T,
) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.request-part", mvc.NewRestController(
				"parts",
				mvc.POST(
					"/parts",
					mvc.JSON(
						http.StatusCreated,
						func(ctx *arkweb.Context) (map[string]string, error) {
							metadata, err := mvc.ValidatedRequestPartJSON[starterRequestPartMetadata](
								ctx,
								"metadata",
								[]string{"create"},
							)
							if err != nil {
								return nil, err
							}
							file, err := mvc.RequestPart(ctx, "file")
							if err != nil {
								return nil, err
							}
							reader, err := file.Open()
							if err != nil {
								return nil, err
							}
							defer reader.Close()
							body, err := io.ReadAll(reader)
							if err != nil {
								return nil, err
							}
							return map[string]string{
								"name":     metadata.Name,
								"filename": file.SubmittedFileName(),
								"body":     string(body),
							}, nil
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
	invalid := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterJSONRequestPartBody(
			t,
			`{}`,
			"application/vnd.goark.metadata+json",
		)
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/parts",
			strings.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusUnprocessableEntity)
	if invalid.header.Get("Content-Type") != "application/problem+json" ||
		!strings.Contains(invalid.body, `"path":"name"`) {
		t.Fatalf(
			"validation response = %d %#v %q, want name violation",
			http.StatusUnprocessableEntity,
			invalid.header,
			invalid.body,
		)
	}

	unsupported := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterJSONRequestPartBody(t, `plain`, "text/plain")
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/parts",
			strings.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusUnsupportedMediaType)
	if unsupported.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf(
			"unsupported Content-Type = %q, want problem json",
			unsupported.header.Get("Content-Type"),
		)
	}

	created := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterJSONRequestPartBody(
			t,
			`{"name":"avatar"}`,
			"application/vnd.goark.metadata+json",
		)
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/parts",
			strings.NewReader(body),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusCreated)
	var payload map[string]string
	if err := arkjson.Unmarshal(nil, []byte(created.body), &payload); err != nil {
		t.Fatalf("created JSON invalid: %v", err)
	}
	if payload["name"] != "avatar" || payload["filename"] != "profile.txt" ||
		payload["body"] != "hello" {
		t.Fatalf("created payload = %#v", payload)
	}
}
func TestAutoConfigure_whenMVCResponseStatusExists_shouldApplyDefaultStatus(t *testing.T) {
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
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.response-status", mvc.NewRestController(
				"api",
				mvc.GET(
					"/api/jobs/accepted",
					mvc.ResponseStatus(
						http.StatusAccepted,
						mvc.Return(0, func(_ *arkweb.Context) (string, error) {
							return "accepted", nil
						}),
					),
				),
				mvc.GET(
					"/api/jobs/empty",
					mvc.ResponseStatus(
						http.StatusAccepted,
						mvc.NoContent(func(_ *arkweb.Context) error {
							return nil
						}),
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
	bodySnapshot := requestUntilStatusSnapshot(
		t,
		serverURL+"/api/jobs/accepted",
		http.StatusAccepted,
	)
	if bodySnapshot.body != "accepted" {
		t.Fatalf("response status body = %q, want accepted", bodySnapshot.body)
	}
	emptySnapshot := requestUntilStatusSnapshot(t, serverURL+"/api/jobs/empty", http.StatusAccepted)
	if emptySnapshot.body != "" {
		t.Fatalf("response status empty body = %q, want empty", emptySnapshot.body)
	}
}

func TestAutoConfigure_whenHandlerReturnsStatusError_shouldUseProblemDetailsStatus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	cause := errors.New("internal quota bucket")

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.status-error", mvc.NewController(
				"errors",
				mvc.GET(
					"/limited",
					mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
						return nil, goweb.NewStatusError(
							http.StatusTooManyRequests,
							"rate limited",
							cause,
						)
					}),
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
		starterServerURL(t, app)+"/limited",
		http.StatusTooManyRequests,
	)
	if snapshot.header.Get("Content-Type") != problem.MediaType {
		t.Fatalf("Content-Type = %q, want problem json", snapshot.header.Get("Content-Type"))
	}
	if !strings.Contains(snapshot.body, `"status":429`) ||
		!strings.Contains(snapshot.body, `"detail":"rate limited"`) ||
		strings.Contains(snapshot.body, cause.Error()) {
		t.Fatalf("problem body = %q", snapshot.body)
	}
}

func TestAutoConfigure_whenErrorEndpointRequestedDirectly_shouldReturnProblemDetails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
    error:
      path: /error
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/error",
		http.StatusInternalServerError,
	)
	if snapshot.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("content type = %q, want problem json", snapshot.header.Get("Content-Type"))
	}
	if !strings.Contains(snapshot.body, `"title":"Internal Server Error"`) ||
		!strings.Contains(snapshot.body, `"instance":"/error"`) {
		t.Fatalf("error endpoint body = %q", snapshot.body)
	}
}
func (starterResponseAdviceConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return gbcweb.RegisterResponseAdvice(
		config.Registry(),
		"testStarterResponseAdvice",
		arkweb.ResponseAdviceFunc(func(_ *arkweb.Context, _ arkweb.Result) (arkweb.Result, error) {
			return arkweb.JSON(http.StatusAccepted, map[string]string{"name": "goark-advised"}), nil
		}),
		container.WithOrder(-100),
	)
}

type starterParameterMapPayload struct {
	Path         map[string]string   `json:"path"`
	Params       map[string]string   `json:"params"`
	ParamValues  map[string][]string `json:"paramValues"`
	Headers      map[string]string   `json:"headers"`
	HeaderValues map[string][]string `json:"headerValues"`
	Cookies      map[string]string   `json:"cookies"`
	CookieValues map[string][]string `json:"cookieValues"`
}
type starterIndexedSearchOwner struct {
	Name    string   `form:"name"`
	Level   int      `form:"level"`
	Aliases []string `form:"aliases"`
}

type starterConversionPayload struct {
	Page   int    `json:"page"`
	Tenant string `json:"tenant"`
}

type starterUploadRequest struct {
	Title string                `form:"title"`
	File  servletmultipart.Part `             multipart:"file"`
}

func (starterResponseAdviceConfiguration) Name() string {
	return "test.web.response-advice"
}

func (starterWebFeaturesConfiguration) Order() int {
	return 0
}
