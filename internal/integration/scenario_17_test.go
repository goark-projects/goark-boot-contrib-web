package gbcweb_test

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenConditionalRequestMatches_shouldReturnNotModified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	modified := time.Date(2026, time.August, 29, 8, 30, 0, 900, time.FixedZone("CST", 8*60*60))

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.conditional", mvc.NewController(
				"conditional",
				mvc.GET(
					"/reports/summary",
					mvc.Handler(func(ctx *arkweb.Context) (arkweb.Result, error) {
						if goweb.CheckNotModified(ctx, "summary-v1", modified) {
							return nil, nil
						}
						return goweb.OK(map[string]string{"state": "fresh"}).
							WithETag("summary-v1").
							WithLastModified(modified), nil
					}),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	first := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/reports/summary",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)
	if first.header.Get("ETag") != `"summary-v1"` ||
		first.header.Get("Last-Modified") != "Sat, 29 Aug 2026 00:30:00 GMT" ||
		first.body != `{"state":"fresh"}` {
		t.Fatalf("initial response = %#v body %q", first.header, first.body)
	}

	second := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/reports/summary",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		request.Header.Set("If-None-Match", first.header.Get("ETag"))
		return request, nil
	}, http.StatusNotModified)
	if second.header.Get("ETag") != `"summary-v1"` ||
		second.header.Get("Last-Modified") != "Sat, 29 Aug 2026 00:30:00 GMT" ||
		second.body != "" {
		t.Fatalf("not modified response = %#v body %q", second.header, second.body)
	}
}

func TestAutoConfigure_whenProducesConditionsOverlap_shouldPreserveSelectedProducesThroughArkhos(
	t *testing.T,
) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
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
			mvc.NewConfiguration("test.mvc.produces-condition-dispatch", mvc.NewRestController(
				"producesDispatch",
				mvc.GET(
					"/produces-dispatch",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterRequestConditionPayload, error) {
							produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
							return starterRequestConditionPayload{Produces: produces.(string)}, nil
						},
					),
					mvc.WithProduces("application/json"),
				),
				mvc.GET(
					"/produces-dispatch",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterRequestConditionPayload, error) {
							produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
							return starterRequestConditionPayload{Produces: produces.(string)}, nil
						},
					),
					mvc.WithProduces("text/plain"),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	requestURL := starterServerURL(t, app) + "/produces-dispatch"
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", "application/json, text/plain")
		return request, nil
	}, http.StatusOK)
	if got := snapshot.header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var payload starterRequestConditionPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Produces != "application/json" {
		t.Fatalf("payload = %#v, want selected application/json produces", payload)
	}
}

func TestAutoConfigure_whenAdviceModelInitializerExists_shouldRenderTemplateModel(
	t *testing.T,
) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates")
	mkdir(t, templateDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(templateDir, "dashboard.html"), "<h1>{{.AppName}} {{.Title}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	advice := mvc.NewControllerAdvice("test.mvc.global-model").WithModelAttributes(
		mvc.ModelAttributeValue("AppName", func(_ *arkweb.Context) (string, error) {
			return "Goark", nil
		}),
		mvc.ModelAttributeInitializerFunc(func(_ *arkweb.Context, model mvc.Model) (mvc.Model, error) {
			return model.AddAttribute("Title", "Default"), nil
		}),
	)
	controller := mvc.NewController("pages",
		mvc.GET("/dashboard", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
			return mvc.NewModelAndView(
				"dashboard",
				mvc.NewModel().AddAttribute("Title", "Dashboard"),
			), nil
		})),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.advice-model-attribute", controller).
				WithControllerAdvices(advice),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/dashboard", http.StatusOK)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>Goark Dashboard</h1>" {
		t.Fatalf("dashboard body = %q, want advice initialized model", snapshot.body)
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
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/locale/change?lang=ko-KR",
			nil,
		)
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
	if !payload.OK || payload.Locale != "ko-KR" || payload.Language != "ko" ||
		payload.Region != "KR" ||
		payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want locale changed by interceptor", payload)
	}
}

func starterNamedRequestPartsBody(t testing.TB) (string, string) {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	for _, item := range []struct {
		name     string
		filename string
		body     string
	}{
		{name: "file", filename: "first.txt", body: "first"},
		{name: "avatar", filename: "ignored.txt", body: "ignored"},
		{name: "file", filename: "second.txt", body: "second"},
	} {
		part, err := writer.CreateFormFile(item.name, item.filename)
		if err != nil {
			t.Fatalf("CreateFormFile failed: %v", err)
		}
		if _, err := io.WriteString(part, item.body); err != nil {
			t.Fatalf("write part failed: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	return body.String(), writer.FormDataContentType()
}

type starterRequestEntityPayload struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Method        string `json:"method"`
	URL           string `json:"url"`
	Path          string `json:"path"`
	TraceID       string `json:"traceId"`
	ContentLength int64  `json:"contentLength"`
	HasBody       bool   `json:"hasBody"`
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q failed: %v", path, err)
	}
}

type starterMappedSearchPayload struct {
	Level  int      `json:"level"`
	Roles  []string `json:"roles"`
	Groups []string `json:"groups"`
}

type starterBinderProfile struct {
	Email string `form:"email" json:"email"`
	Admin bool   `form:"admin" json:"admin"`
}

func (starterErrorMapperConfiguration) Name() string {
	return "test.web.error-mapper"
}

func (starterMessageConverterConfiguration) Name() string {
	return "test.web.message-converter"
}

type starterRequestConditionPayload struct {
	Produces string `json:"produces"`
}

type starterRejectingValidator struct{}
