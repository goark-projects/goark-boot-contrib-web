package gbcweb_test

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/web/mvc"
	mvcview "goark.dev/goark/web/mvc/view"
	gowebstatic "goark.dev/goark/web/static"
)

var errStarterMapped = errors.New("starter mapped")

type starterForwardedPayload struct {
	URL    string `json:"url"`
	Remote string `json:"remote"`
}

type starterGroupedCreateRequest struct {
	Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" arkarta:"required"`
}

type starterAdviceError struct {
	id string
}

func (e *starterAdviceError) Error() string {
	return "starter advice " + e.id
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

func TestAutoConfigure_whenDefaultResourceStaticExists_shouldServeStaticResources(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	staticDir := filepath.Join(resource, "static")
	publicDir := filepath.Join(resource, "public")
	mkdir(t, staticDir)
	mkdir(t, publicDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(staticDir, "app.txt"), "resource static")
	writeFile(t, filepath.Join(publicDir, "public.txt"), "resource public")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	if body := requestUntilOK(t, serverURL+"/static/app.txt"); body != "resource static" {
		t.Fatalf("static body = %q, want resource static", body)
	}
	if body := requestUntilOK(t, serverURL+"/static/public.txt"); body != "resource public" {
		t.Fatalf("public static body = %q, want resource public", body)
	}
}

func TestAutoConfigure_whenStaticResourcesConfigured_shouldServeConfiguredLocationAndPattern(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	publicDir := filepath.Join(root, "public")
	mkdir(t, resource)
	mkdir(t, publicDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
    resources:
      static-locations: public
      static:
        welcome-files: index.html
      chain:
        strategy:
          content:
            enabled: true
          fixed:
            version: v1
      cache:
        cachecontrol:
          max-age: 1h
  mvc:
    static-path-pattern: /assets/*
`)
	writeFile(t, filepath.Join(publicDir, "app.txt"), "configured static")
	writeFile(t, filepath.Join(publicDir, "index.html"), "configured index")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	staticSnapshot := requestUntilStatusSnapshot(t, serverURL+"/assets/app.txt", http.StatusOK)
	if staticSnapshot.body != "configured static" {
		t.Fatalf("configured static body = %q", staticSnapshot.body)
	}
	if got := staticSnapshot.header.Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("Cache-Control = %q, want public max-age", got)
	}
	if body := requestUntilOK(t, serverURL+"/assets/"); body != "configured index" {
		t.Fatalf("configured welcome body = %q", body)
	}
	versioned, err := gowebstatic.ContentVersionPath(t.Context(), os.DirFS(publicDir), "app.txt")
	if err != nil {
		t.Fatalf("ContentVersionPath failed: %v", err)
	}
	versioned, err = gowebstatic.FixedVersionPath("v1", versioned)
	if err != nil {
		t.Fatalf("FixedVersionPath failed: %v", err)
	}
	if body := requestUntilOK(t, serverURL+"/assets/"+versioned); body != "configured static" {
		t.Fatalf("versioned static body = %q", body)
	}
	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	provider, err := goark.Get[gowebstatic.ResourceURLProvider](t.Context(), appContext, gbcweb.BeanNameStaticResourceURLProvider)
	if err != nil {
		t.Fatalf("resolve static resource url provider failed: %v", err)
	}
	resourceURL, err := provider.URL(t.Context(), "app.txt")
	if err != nil {
		t.Fatalf("provider URL failed: %v", err)
	}
	if !strings.HasPrefix(resourceURL, "/assets/v1/app-") || !strings.HasSuffix(resourceURL, ".txt") {
		t.Fatalf("resource url = %q, want versioned assets URL", resourceURL)
	}
	if body := requestUntilOK(t, serverURL+resourceURL); body != "configured static" {
		t.Fatalf("provider static body = %q", body)
	}
}

func TestAutoConfigure_whenDefaultTemplateExists_shouldRenderMVCView(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "home.html"), "<h1>{{.Title}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.view", mvc.NewController("views",
			mvc.GET("/home", mvc.Handler(func(_ *arkweb.Context) (arkweb.Result, error) {
				return mvcview.Render("home", map[string]string{"Title": "Goark"}), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/home", http.StatusOK)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>Goark</h1>" {
		t.Fatalf("view body = %q, want rendered html", snapshot.body)
	}
}

func TestAutoConfigure_whenMVCViewControllerExists_shouldRenderTemplate(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "ready.html"), "<h1>ready</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.view-controller", mvc.NewController("viewControllers",
			mvc.ViewController("/ready", "ready", mvc.WithViewControllerStatus(http.StatusAccepted)),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/ready", http.StatusAccepted)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>ready</h1>" {
		t.Fatalf("view body = %q, want rendered html", snapshot.body)
	}
}

func TestAutoConfigure_whenMVCModelViewExists_shouldRenderTemplateModel(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates")
	mkdir(t, filepath.Join(templateDir, "reports"))
	mkdir(t, filepath.Join(templateDir, "pages"))
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(templateDir, "reports", "summary.html"), "<h1>{{.Title}}</h1>")
	writeFile(t, filepath.Join(templateDir, "pages", "detail.html"), "<h1>{{.Title}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-view", mvc.NewController("pages",
			mvc.GET("/reports/summary.html", mvc.Return(0, func(_ *arkweb.Context) (mvc.Model, error) {
				return mvc.NewModel().AddAttribute("Title", "Summary"), nil
			})),
			mvc.GET("/pages/42", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
				model := mvc.NewModel().AddAttribute("Title", "Detail")
				return mvc.NewModelAndView("pages/detail", model, mvc.WithViewStatus(http.StatusAccepted)), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	inferred := requestUntilStatusSnapshot(t, serverURL+"/reports/summary.html", http.StatusOK)
	if inferred.body != "<h1>Summary</h1>" {
		t.Fatalf("inferred model view body = %q, want summary", inferred.body)
	}
	explicit := requestUntilStatusSnapshot(t, serverURL+"/pages/42", http.StatusAccepted)
	if explicit.body != "<h1>Detail</h1>" {
		t.Fatalf("explicit model and view body = %q, want detail", explicit.body)
	}
}
