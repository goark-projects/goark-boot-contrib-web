package gbcweb_test

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/static"
)

type starterUploadRequest struct {
	Title string                `form:"title"`
	File  servletmultipart.Part `multipart:"file"`
}

func TestAutoConfigure_whenWebFeatureContributorsExist_shouldServeThroughArkhos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			starterWebFeaturesConfiguration{},
			mvc.NewConfiguration("test.web.features.mvc", mvc.NewController("files",
				mvc.POST("/uploads", mvc.BindMultipart(http.StatusCreated, func(_ *arkweb.Context, input starterUploadRequest) (map[string]string, error) {
					file, err := input.File.Open()
					if err != nil {
						return nil, err
					}
					defer file.Close()
					data, err := io.ReadAll(file)
					if err != nil {
						return nil, err
					}
					return map[string]string{
						"title":    input.Title,
						"filename": input.File.SubmittedFileName(),
						"body":     string(data),
					}, nil
				})),
				mvc.GET("/reports/today", mvc.Handler(func(_ *arkweb.Context) (arkweb.Result, error) {
					return goweb.Attachment("today.csv", strings.NewReader("id,name\n1,goark\n"),
						goweb.WithDownloadContentType("text/csv"),
						goweb.WithDownloadContentLength(16),
					), nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	staticSnapshot := requestUntilStatusSnapshot(t, serverURL+"/assets/app.txt", http.StatusOK)
	if staticSnapshot.body != "starter static" {
		t.Fatalf("static body = %q", staticSnapshot.body)
	}
	if got := staticSnapshot.header.Get("X-Starter-Filter"); got != "hit" {
		t.Fatalf("static X-Starter-Filter = %q, want hit", got)
	}

	uploadSnapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterMultipartBody(t)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/uploads", strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusCreated)
	if uploadSnapshot.header.Get("X-Starter-Filter") != "hit" || uploadSnapshot.header.Get("X-Starter-Interceptor") != "hit" {
		t.Fatalf("upload headers = %#v", uploadSnapshot.header)
	}
	var uploadPayload map[string]string
	if err := arkjson.Unmarshal(nil, []byte(uploadSnapshot.body), &uploadPayload); err != nil {
		t.Fatalf("upload json invalid: %v", err)
	}
	if uploadPayload["title"] != "avatar" || uploadPayload["filename"] != "profile.txt" || uploadPayload["body"] != "hello" {
		t.Fatalf("upload payload = %#v", uploadPayload)
	}

	downloadSnapshot := requestUntilStatusSnapshot(t, serverURL+"/reports/today", http.StatusOK)
	if downloadSnapshot.body != "id,name\n1,goark\n" {
		t.Fatalf("download body = %q", downloadSnapshot.body)
	}
	if got := downloadSnapshot.header.Get("Content-Type"); got != "text/csv" {
		t.Fatalf("download Content-Type = %q, want text/csv", got)
	}
	if got := downloadSnapshot.header.Get("Content-Disposition"); got != `attachment; filename=today.csv` {
		t.Fatalf("download Content-Disposition = %q", got)
	}
}

type starterWebFeaturesConfiguration struct{}

func (starterWebFeaturesConfiguration) Name() string {
	return "test.web.features"
}

func (starterWebFeaturesConfiguration) Order() int {
	return 0
}

func (c starterWebFeaturesConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterWebFeaturesConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	if err := goweb.RegisterInterceptor(config.Registry(), "testInterceptor", goweb.InterceptorFunc(func(ctx *arkweb.Context, next arkweb.Handler) (arkweb.Result, error) {
		ctx.Response().Header().Set("X-Starter-Interceptor", "hit")
		return next.Handle(ctx)
	})); err != nil {
		return err
	}
	if err := goweb.RegisterFilter(config.Registry(), "testFilter", servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		res.Header().Set("X-Starter-Filter", "hit")
		return chain.Next(ctx, req, res)
	})); err != nil {
		return err
	}
	return static.Register(config.Registry(), "testStaticResources", "/assets/*", fstest.MapFS{
		"app.txt": &fstest.MapFile{
			Data:    []byte("starter static"),
			Mode:    0o644,
			ModTime: time.Unix(10, 0),
		},
	})
}

func starterServerURL(t *testing.T, app *boot.Application) string {
	t.Helper()
	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	return server.URL()
}

func requestUntilStatusWith(t *testing.T, build func() (*http.Request, error), statusCode int) responseSnapshot {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		request, err := build()
		if err != nil {
			t.Fatalf("build request failed: %v", err)
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
			t.Fatalf("status = %d, want %d, body = %q", response.StatusCode, statusCode, string(body))
		}
		if time.Now().After(deadline) {
			t.Fatalf("request did not succeed before deadline: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func starterMultipartBody(t *testing.T) (string, string) {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "avatar"); err != nil {
		t.Fatalf("WriteField failed: %v", err)
	}
	part, err := writer.CreateFormFile("file", "profile.txt")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := io.WriteString(part, "hello"); err != nil {
		t.Fatalf("write part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	return body.String(), writer.FormDataContentType()
}
