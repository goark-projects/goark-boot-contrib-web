package gbcweb_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	webclient "goark.dev/goark/web/client"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenHTTPClientConfigured_shouldRegisterBuilderAndClient(t *testing.T) {
	serverErrors := make(chan error, 2)
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-App") != "boot" {
			failHTTPServer(serverErrors, writer, "X-App = %q, want boot", request.Header.Get("X-App"))
			return
		}
		switch request.URL.Path {
		case "/api/ping":
			session, err := request.Cookie("sid")
			if err != nil || session.Value != "abc" {
				failHTTPServer(serverErrors, writer, "sid cookie = %#v err %v", session, err)
				return
			}
			_, _ = io.WriteString(writer, `{"status":"UP"}`)
		case "/api/builder":
			if request.Header.Get("X-Builder") != "yes" {
				failHTTPServer(serverErrors, writer, "X-Builder = %q, want yes", request.Header.Get("X-Builder"))
				return
			}
			_, _ = io.WriteString(writer, "builder")
		case "/api/upload":
			if err := request.ParseMultipartForm(1 << 20); err != nil {
				failHTTPServer(serverErrors, writer, "parse multipart failed: %v", err)
				return
			}
			file, header, err := request.FormFile("file")
			if err != nil {
				failHTTPServer(serverErrors, writer, "form file failed: %v", err)
				return
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			if err != nil {
				failHTTPServer(serverErrors, writer, "read file failed: %v", err)
				return
			}
			if request.FormValue("title") != "avatar" || header.Filename != "profile.txt" || string(body) != "hello" {
				failHTTPServer(serverErrors, writer, "multipart = %q %q %q", request.FormValue("title"), header.Filename, string(body))
				return
			}
			writer.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(writer, "uploaded")
		default:
			failHTTPServer(serverErrors, writer, "path = %q", request.URL.Path)
		}
	}))
	defer api.Close()

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
			gbcweb.WithHTTPClientBaseURL(api.URL+"/api"),
			gbcweb.WithHTTPClientTimeout(2*time.Second),
			gbcweb.WithHTTPClientMaxResponseBytes(64),
			gbcweb.WithHTTPClientDefaultHeader("X-App", "boot"),
		)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	defaultClient, err := goark.Get[*webclient.Client](t.Context(), appContext, gbcweb.BeanNameHTTPClient)
	if err != nil {
		t.Fatalf("resolve http client failed: %v", err)
	}
	response, err := defaultClient.Get(t.Context(), "/ping", webclient.WithCookieValue("sid", "abc"))
	if err != nil {
		t.Fatalf("default client get failed: %v", err)
	}
	var payload map[string]string
	if err := response.DecodeJSON(&payload); err != nil {
		t.Fatalf("decode client response failed: %v", err)
	}
	if payload["status"] != "UP" {
		t.Fatalf("payload = %#v", payload)
	}

	builder, err := goark.Get[*webclient.Builder](t.Context(), appContext, gbcweb.BeanNameHTTPClientBuilder)
	if err != nil {
		t.Fatalf("resolve http client builder failed: %v", err)
	}
	derivedClient, err := builder.DefaultHeader("X-Builder", "yes").Build()
	if err != nil {
		t.Fatalf("derived client build failed: %v", err)
	}
	derivedResponse, err := derivedClient.Get(t.Context(), "/builder")
	if err != nil {
		t.Fatalf("derived client get failed: %v", err)
	}
	if derivedResponse.BodyString() != "builder" {
		t.Fatalf("derived body = %q, want builder", derivedResponse.BodyString())
	}
	uploadResponse, err := defaultClient.Post(t.Context(), "/upload",
		webclient.WithMultipartFields(map[string]string{"title": "avatar"}, webclient.MultipartFile{
			FieldName:   "file",
			FileName:    "profile.txt",
			ContentType: "text/plain",
			Body:        strings.NewReader("hello"),
		}),
	)
	if err != nil {
		t.Fatalf("multipart client post failed: %v", err)
	}
	if uploadResponse.StatusCode() != http.StatusCreated || uploadResponse.BodyString() != "uploaded" {
		t.Fatalf("upload response = %d %q, want 201 uploaded", uploadResponse.StatusCode(), uploadResponse.BodyString())
	}
	assertNoHTTPServerError(t, serverErrors)
}

func TestAutoConfigure_whenHTTPClientBuilderCustomizersExist_shouldApplyInOrder(t *testing.T) {
	serverErrors := make(chan error, 1)
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/customized" {
			failHTTPServer(serverErrors, writer, "path = %q, want /customized", request.URL.Path)
			return
		}
		values := request.Header.Values("X-Chain")
		if len(values) != 2 || values[0] != "first" || values[1] != "second" {
			failHTTPServer(serverErrors, writer, "X-Chain = %#v, want first then second", values)
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
	}))
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
	defaultClient, err := goark.Get[*webclient.Client](t.Context(), appContext, gbcweb.BeanNameHTTPClient)
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.problem", mvc.NewController("problem",
			mvc.GET("/problem", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
				return nil, errors.New("internal secret")
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/problem", http.StatusInternalServerError)
	if snapshot.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("content type = %q, want problem json", snapshot.header.Get("Content-Type"))
	}
	if !strings.Contains(snapshot.body, `"status":500`) ||
		!strings.Contains(snapshot.body, `"instance":"/problem"`) ||
		strings.Contains(snapshot.body, "internal secret") {
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

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/error", http.StatusInternalServerError)
	if snapshot.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("content type = %q, want problem json", snapshot.header.Get("Content-Type"))
	}
	if !strings.Contains(snapshot.body, `"title":"Internal Server Error"`) ||
		!strings.Contains(snapshot.body, `"instance":"/error"`) {
		t.Fatalf("error endpoint body = %q", snapshot.body)
	}
}

func TestAutoConfigure_whenWebErrorMapperConfigurerExists_shouldApplyMapper(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc", mvc.NewController("errors",
				mvc.GET("/errors", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return nil, errStarterMapped
				})),
			)),
			starterErrorMapperConfiguration{},
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	body := requestUntilStatus(t, server.URL()+"/errors", http.StatusConflict)
	if body != "mapped" {
		t.Fatalf("body = %q, want mapped", body)
	}
}

func TestAutoConfigure_whenDependentErrorMapperConfigurerExists_shouldApplyBeforeProblemDetails(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.dependent-error", mvc.NewController("dependent-errors",
				mvc.GET("/dependent-errors", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return nil, errStarterMapped
				})),
			)),
			starterDependentErrorMapperConfiguration{},
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	body := requestUntilStatus(t, starterServerURL(t, app)+"/dependent-errors", http.StatusConflict)
	if body != "mapped" {
		t.Fatalf("body = %q, want mapped", body)
	}
}
