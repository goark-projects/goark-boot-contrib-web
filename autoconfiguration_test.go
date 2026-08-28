package gbcweb_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

var errStarterMapped = errors.New("starter mapped")

func TestAutoConfigure_whenMVCControllerExists_shouldServeRequestWithArkhos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc", mvc.NewController("health",
			mvc.GET("/healthz", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
				return map[string]string{"status": "UP"}, nil
			})),
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
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	body := requestUntilOK(t, server.URL()+"/healthz")
	if body != `{"status":"UP"}`+"\n" {
		t.Fatalf("body = %q, want health json", body)
	}
}

func TestAutoConfigure_whenResponseEntityRouteExists_shouldPreserveStatusHeadersAndBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.entity", mvc.NewController("jobs",
			mvc.GET("/jobs/1", mvc.Entity(func(_ *arkweb.Context) (goweb.ResponseEntity[map[string]string], error) {
				return goweb.Status(http.StatusAccepted, map[string]string{"state": "queued"}).
					WithHeader("X-Starter-Entity", "true"), nil
			})),
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
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](t.Context(), appContext, gbcarkhos.BeanNameServer)
	if err != nil {
		t.Fatalf("resolve embedded server failed: %v", err)
	}
	snapshot := requestUntilStatusSnapshot(t, server.URL()+"/jobs/1", http.StatusAccepted)
	if snapshot.header.Get("X-Starter-Entity") != "true" {
		t.Fatalf("X-Starter-Entity = %q, want true", snapshot.header.Get("X-Starter-Entity"))
	}
	if snapshot.body != `{"state":"queued"}`+"\n" {
		t.Fatalf("body = %q, want entity json", snapshot.body)
	}
}

func TestAutoConfigure_whenWebErrorMapperConfigurerExists_shouldApplyMapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
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

func TestAutoConfigure_whenRegisteredTwice_shouldBackOffExistingConfigurations(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(
			gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0"))),
			gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0"))),
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
	assertConfigurationCount(t, appContext.Configurations(), "goark.boot.contrib.web.configuration", 1)
	assertConfigurationCount(t, appContext.Configurations(), "goark.boot.contrib.arkhos.configuration", 1)
}

func requestUntilOK(t *testing.T, target string) string {
	return requestUntilStatus(t, target, http.StatusOK)
}

func requestUntilStatus(t *testing.T, target string, statusCode int) string {
	return requestUntilStatusSnapshot(t, target, statusCode).body
}

type responseSnapshot struct {
	body   string
	header http.Header
}

func requestUntilStatusSnapshot(t *testing.T, target string, statusCode int) responseSnapshot {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, err := client.Get(target)
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
			t.Fatalf("GET %s did not succeed before deadline: %v", target, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type starterErrorMapperConfiguration struct{}

func (starterErrorMapperConfiguration) Name() string {
	return "test.web.error-mapper"
}

func (starterErrorMapperConfiguration) Order() int {
	return 0
}

func (c starterErrorMapperConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterErrorMapperConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return goweb.RegisterErrorMapper(config.Registry(), "testErrorMapper", goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
		if errors.Is(err, errStarterMapped) {
			return arkweb.Text(http.StatusConflict, "mapped")
		}
		return nil
	}))
}

func closeApp(t *testing.T, app *boot.Application) {
	t.Helper()
	if err := app.Close(t.Context()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q failed: %v", path, err)
	}
}

func assertConfigurationCount(t *testing.T, descriptors []goark.ConfigurationDescriptor, name string, want int) {
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
