package gbcweb_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	webclient "goark.dev/goark/web/client"
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

func TestAutoConfigure_whenMVCModelAttributeInitializerExists_shouldRenderTemplateModel(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "home.html"), "<h1>{{.AppName}}</h1>")
	writeFile(t, filepath.Join(templateDir, "dashboard.html"), "<h1>{{.AppName}} {{.Title}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	controller := mvc.NewController("pages",
		mvc.GET("/home", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "home", nil
		})),
		mvc.GET("/dashboard", mvc.Return(0, func(_ *arkweb.Context) (mvc.Model, error) {
			return mvc.NewModel().AddAttribute("Title", "Dashboard"), nil
		})),
	).WithModelAttributes(
		mvc.ModelAttributeValue("AppName", func(_ *arkweb.Context) (string, error) {
			return "Goark", nil
		}),
		mvc.ModelAttributeInitializerFunc(func(_ *arkweb.Context, model mvc.Model) (mvc.Model, error) {
			return model.AddAttribute("Title", "Default"), nil
		}),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-attribute", controller)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	home := requestUntilStatusSnapshot(t, serverURL+"/home", http.StatusOK)
	if home.body != "<h1>Goark</h1>" {
		t.Fatalf("home body = %q, want initialized model", home.body)
	}
	dashboard := requestUntilStatusSnapshot(t, serverURL+"/dashboard", http.StatusOK)
	if dashboard.body != "<h1>Goark Dashboard</h1>" {
		t.Fatalf("dashboard body = %q, want merged model", dashboard.body)
	}
}

func TestAutoConfigure_whenControllerAdviceModelAttributeInitializerExists_shouldRenderTemplateModel(t *testing.T) {
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
			return mvc.NewModelAndView("dashboard", mvc.NewModel().AddAttribute("Title", "Dashboard")), nil
		})),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.advice-model-attribute", controller).WithControllerAdvices(advice)),
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

func TestAutoConfigure_whenControllerAndRestControllerReturnValuesExist_shouldUseDefaultStrategies(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "home.html"), "<h1>home</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.controller-kind",
			mvc.NewController("pages",
				mvc.GET("/home", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
					return "home", nil
				})),
			),
			mvc.NewRestController("api",
				mvc.GET("/api/status", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
					return "UP", nil
				})),
			),
		)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	viewSnapshot := requestUntilStatusSnapshot(t, serverURL+"/home", http.StatusOK)
	if viewSnapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("view Content-Type = %q, want html", viewSnapshot.header.Get("Content-Type"))
	}
	if viewSnapshot.body != "<h1>home</h1>" {
		t.Fatalf("view body = %q, want rendered view", viewSnapshot.body)
	}
	restSnapshot := requestUntilStatusSnapshot(t, serverURL+"/api/status", http.StatusOK)
	if restSnapshot.header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("rest Content-Type = %q, want text/plain", restSnapshot.header.Get("Content-Type"))
	}
	if restSnapshot.body != "UP" {
		t.Fatalf("rest body = %q, want raw response body", restSnapshot.body)
	}
}

func TestAutoConfigure_whenControllerReturnsRedirectViewName_shouldWriteRedirect(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.redirect", mvc.NewController("redirects",
			mvc.GET("/accounts", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
				return "redirect:/signin", nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWithClient(t, http.Client{
		Timeout: time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/accounts", nil)
	}, http.StatusFound)
	if got := snapshot.header.Get("Location"); got != "/signin" {
		t.Fatalf("Location = %q, want /signin", got)
	}
	if snapshot.body != "" {
		t.Fatalf("body = %q, want empty", snapshot.body)
	}
}

func TestAutoConfigure_whenModelViewRedirectHasAttributes_shouldExpandLocation(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.redirect-attributes", mvc.NewController("redirects",
			mvc.GET("/accounts", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
				model := mvc.NewModel().
					AddAttribute("id", "a/b").
					AddAttribute("page", 2).
					AddAttribute("tab", "security")
				return mvc.NewModelAndView("redirect:/users/{id}", model, mvc.WithViewStatus(http.StatusSeeOther)), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWithClient(t, http.Client{
		Timeout: time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/accounts", nil)
	}, http.StatusSeeOther)
	if got := snapshot.header.Get("Location"); got != "/users/a%2Fb?page=2&tab=security" {
		t.Fatalf("Location = %q, want expanded redirect", got)
	}
}

func TestAutoConfigure_whenControllerReturnsForwardViewName_shouldDispatchTarget(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.forward", mvc.NewController("forwards",
			mvc.GET("/source", mvc.Return(0, func(_ *arkweb.Context) (string, error) {
				return "forward:/target?from=source", nil
			})),
			mvc.GET("/target", mvc.ResponseBody(http.StatusAccepted, func(ctx *arkweb.Context) (string, error) {
				forwardURI, ok := ctx.Request().Attribute(servlet.AttributeForwardRequestURI)
				uri, ok := forwardURI.(string)
				if !ok {
					return "missing-forward-attribute", nil
				}
				return ctx.QueryValue("from") + ":" + uri, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/source", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", "text/plain")
		return request, nil
	}, http.StatusAccepted)
	if snapshot.header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "source:/source" {
		t.Fatalf("body = %q, want forwarded target body", snapshot.body)
	}
}

func TestAutoConfigure_whenControllerResponseBodyExists_shouldBypassViewResolution(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "status.html"), "<h1>view</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.response-body",
			mvc.NewController("status",
				mvc.GET("/status", mvc.ResponseBody(http.StatusOK, func(_ *arkweb.Context) (string, error) {
					return "UP", nil
				})),
			),
		)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/status", http.StatusOK)
	if snapshot.header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "UP" {
		t.Fatalf("body = %q, want raw response body", snapshot.body)
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.response-status", mvc.NewRestController("api",
			mvc.GET("/api/jobs/accepted", mvc.ResponseStatus(http.StatusAccepted, mvc.Return(0, func(_ *arkweb.Context) (string, error) {
				return "accepted", nil
			}))),
			mvc.GET("/api/jobs/empty", mvc.ResponseStatus(http.StatusAccepted, mvc.NoContent(func(_ *arkweb.Context) error {
				return nil
			}))),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	bodySnapshot := requestUntilStatusSnapshot(t, serverURL+"/api/jobs/accepted", http.StatusAccepted)
	if bodySnapshot.body != "accepted" {
		t.Fatalf("response status body = %q, want accepted", bodySnapshot.body)
	}
	emptySnapshot := requestUntilStatusSnapshot(t, serverURL+"/api/jobs/empty", http.StatusAccepted)
	if emptySnapshot.body != "" {
		t.Fatalf("response status empty body = %q, want empty", emptySnapshot.body)
	}
}

func TestAutoConfigure_whenMVCValidationGroupsExist_shouldUseExplicitGroups(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.validation-groups", mvc.NewRestController("users",
			mvc.POST("/users", mvc.BindJSONGroups(http.StatusCreated, func(_ *arkweb.Context, input starterGroupedCreateRequest) (map[string]string, error) {
				return map[string]string{"name": input.Name}, nil
			}, "create")),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	missingName := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/users", strings.NewReader(`{}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		return request, nil
	}, http.StatusUnprocessableEntity)
	if missingName.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want problem json", missingName.header.Get("Content-Type"))
	}

	created := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/users", strings.NewReader(`{"name":"goark"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		return request, nil
	}, http.StatusCreated)
	if created.body != `{"name":"goark"}` {
		t.Fatalf("body = %q, want created payload", created.body)
	}
}

func TestAutoConfigure_whenRestControllerAdviceExists_shouldMapErrors(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.advice-controller", mvc.NewRestController("users",
				mvc.GET("/advice/{id}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
					id, err := mvc.PathString(ctx, "id")
					if err != nil {
						return nil, err
					}
					return nil, &starterAdviceError{id: id}
				})),
			)),
			mvc.NewRestControllerAdvice("test.mvc.rest-advice",
				mvc.ExceptionReturnAs[*starterAdviceError](http.StatusConflict, func(_ *arkweb.Context, err *starterAdviceError) map[string]string {
					return map[string]string{"id": err.id}
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/advice/42", http.StatusConflict)
	if snapshot.header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != `{"id":"42"}` {
		t.Fatalf("body = %q, want advice payload", snapshot.body)
	}
}

func TestAutoConfigure_whenControllerAdviceValueExists_shouldRenderTemplate(t *testing.T) {
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
	writeFile(t, filepath.Join(templateDir, "missing.html"), "<h1>missing</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.advice-view", mvc.NewController("pages",
				mvc.GET("/pages/missing", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
					return nil, &starterAdviceError{id: "missing"}
				})),
			)),
			mvc.NewControllerAdvice("test.mvc.page-advice",
				mvc.ExceptionReturnAs[*starterAdviceError](http.StatusNotFound, func(_ *arkweb.Context, _ *starterAdviceError) string {
					return "missing"
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/pages/missing", http.StatusNotFound)
	if snapshot.header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want html", snapshot.header.Get("Content-Type"))
	}
	if snapshot.body != "<h1>missing</h1>" {
		t.Fatalf("body = %q, want rendered advice view", snapshot.body)
	}
}

func TestAutoConfigure_whenControllerAdviceResponseEntityExists_shouldPreserveEntity(t *testing.T) {
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
			mvc.NewConfiguration("test.mvc.advice-entity", mvc.NewRestController("users",
				mvc.GET("/users/{id}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
					id, err := mvc.PathString(ctx, "id")
					if err != nil {
						return nil, err
					}
					return nil, &starterAdviceError{id: id}
				})),
			)),
			mvc.NewRestControllerAdvice("test.mvc.entity-advice",
				mvc.ExceptionEntityAs[*starterAdviceError](func(_ *arkweb.Context, err *starterAdviceError) goweb.ResponseEntity[map[string]string] {
					return goweb.Status(http.StatusGone, map[string]string{"id": err.id}).
						WithHeader("X-Starter-Advice", "entity")
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/users/42", http.StatusGone)
	if snapshot.header.Get("X-Starter-Advice") != "entity" {
		t.Fatalf("X-Starter-Advice = %q, want entity", snapshot.header.Get("X-Starter-Advice"))
	}
	if snapshot.body != `{"id":"42"}` {
		t.Fatalf("body = %q, want advice response entity payload", snapshot.body)
	}
}

func TestAutoConfigure_whenResponseEntityRouteExists_shouldPreserveStatusHeadersAndBody(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.entity", mvc.NewController("jobs",
			mvc.POST("/jobs", mvc.Entity(func(ctx *arkweb.Context) (goweb.ResponseEntity[map[string]string], error) {
				return goweb.CreatedFromCurrentRequest(ctx, "/{id}", map[string]string{"id": "42"}, map[string]string{"state": "created"})
			})),
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
	if snapshot.body != `{"state":"queued"}` {
		t.Fatalf("body = %q, want entity json", snapshot.body)
	}
	created := requestUntilStatusSnapshotWithMethod(t, http.MethodPost, server.URL()+"/jobs", http.StatusCreated)
	if got := created.header.Get("Location"); got != server.URL()+"/jobs/42" {
		t.Fatalf("created Location = %q, want current request URI", got)
	}
	if created.body != `{"state":"created"}` {
		t.Fatalf("created body = %q, want entity json", created.body)
	}
}

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

func TestAutoConfigure_whenWebFiltersConfigured_shouldApplyCorsForwardedHeadersAndETag(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
    cors:
      enabled: true
      allowed-origins: https://admin.example.com
      allowed-methods: GET,POST
      allowed-headers: X-Request-ID,Content-Type
      exposed-headers: X-Trace-ID
      allow-credentials: true
      max-age: 5m
    filters:
      forwarded-headers:
        enabled: true
    shallow-etag:
      enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.filters", mvc.NewController("filters",
			mvc.GET("/filters", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterForwardedPayload, error) {
				ctx.Response().Header().Set("X-Trace-ID", "trace-1")
				return starterForwardedPayload{
					URL:    ctx.Request().RequestURL(),
					Remote: ctx.Request().RemoteAddr(),
				}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	preflight := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodOptions, serverURL+"/filters", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		request.Header.Set("Access-Control-Request-Headers", "x-request-id, content-type")
		return request, nil
	}, http.StatusNoContent)
	if preflight.header.Get("Access-Control-Allow-Origin") != "https://admin.example.com" ||
		preflight.header.Get("Access-Control-Allow-Credentials") != "true" ||
		preflight.header.Get("Access-Control-Max-Age") != "300" {
		t.Fatalf("preflight headers = %#v", preflight.header)
	}

	first := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/filters", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "api.example.com")
		request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
		return request, nil
	}, http.StatusOK)
	if first.header.Get("Access-Control-Allow-Origin") != "https://admin.example.com" ||
		first.header.Get("Access-Control-Expose-Headers") != "X-Trace-ID" ||
		first.header.Get("ETag") == "" {
		t.Fatalf("actual headers = %#v", first.header)
	}
	if !strings.Contains(first.body, `"url":"https://api.example.com/filters"`) ||
		!strings.Contains(first.body, `"remote":"203.0.113.7"`) {
		t.Fatalf("forwarded body = %q", first.body)
	}

	second := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/filters", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Origin", "https://admin.example.com")
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "api.example.com")
		request.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
		request.Header.Set("If-None-Match", first.header.Get("ETag"))
		return request, nil
	}, http.StatusNotModified)
	if second.body != "" {
		t.Fatalf("not modified body = %q, want empty", second.body)
	}
}

func TestAutoConfigure_whenHiddenHTTPMethodFilterEnabled_shouldRouteOverriddenMethod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  mvc:
    hiddenmethod:
      filter:
        enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.hidden-method", mvc.NewRestController("items",
			mvc.POST("/items/1", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
				return http.MethodPost, nil
			})),
			mvc.DELETE("/items/1", mvc.Return(http.StatusOK, func(_ *arkweb.Context) (string, error) {
				return http.MethodDelete, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/items/1", strings.NewReader("_method=DELETE"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return request, nil
	}, http.StatusOK)
	if snapshot.body != http.MethodDelete {
		t.Fatalf("body = %q, want DELETE route", snapshot.body)
	}
}

func TestAutoConfigure_whenFormContentFilterEnabled_shouldBindDeleteFormParameters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  mvc:
    formcontent:
      filter:
        enabled: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.form-content", mvc.NewRestController("items",
			mvc.DELETE("/items/1", mvc.Return(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
				return mvc.RequestParamString(ctx, "name")
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, starterServerURL(t, app)+"/items/1", strings.NewReader("name=goark"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return request, nil
	}, http.StatusOK)
	if snapshot.body != "goark" {
		t.Fatalf("body = %q, want form parameter", snapshot.body)
	}
}

func TestAutoConfigure_whenCharacterEncodingFilterEnabled_shouldApplyEncoding(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  servlet:
    encoding:
      charset: UTF-8
      force-response: true
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.encoding", mvc.NewRestController("encoding",
			mvc.POST("/encoding/request", mvc.Return(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
				return ctx.Request().CharacterEncoding(), nil
			})),
			mvc.GET("/encoding/response", mvc.Entity(func(_ *arkweb.Context) (goweb.ResponseEntity[string], error) {
				return goweb.Status(http.StatusOK, "ok").WithContentType("text/plain; charset=iso-8859-1"), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	requestSnapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/encoding/request", strings.NewReader("{}"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		return request, nil
	}, http.StatusOK)
	if !strings.EqualFold(requestSnapshot.body, "UTF-8") {
		t.Fatalf("request encoding body = %q, want UTF-8", requestSnapshot.body)
	}
	responseSnapshot := requestUntilStatusSnapshot(t, serverURL+"/encoding/response", http.StatusOK)
	if got := responseSnapshot.header.Get("Content-Type"); !strings.Contains(strings.ToLower(got), "charset=utf-8") {
		t.Fatalf("response Content-Type = %q, want UTF-8 charset", got)
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
	return requestUntilStatusSnapshotWithMethod(t, http.MethodGet, target, statusCode)
}

func requestUntilStatusSnapshotWithMethod(t *testing.T, method string, target string, statusCode int) responseSnapshot {
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
			t.Fatalf("status = %d, want %d, body = %q", response.StatusCode, statusCode, string(body))
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s %s did not succeed before deadline: %v", method, target, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type starterErrorMapperConfiguration struct{}

type starterErrorMapperDependency struct{}

type starterDependentErrorMapperConfiguration struct{}

func (starterDependentErrorMapperConfiguration) Name() string {
	return "test.web.dependent-error-mapper"
}

func (starterDependentErrorMapperConfiguration) Order() int {
	return 0
}

func (c starterDependentErrorMapperConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterDependentErrorMapperConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	if err := container.RegisterInstance(config.Registry(), "testErrorMapperDependency", &starterErrorMapperDependency{}); err != nil {
		return err
	}
	return container.Register[goweb.Configurer](config.Registry(), "testDependentErrorMapper", func(_ context.Context, _ container.Resolver) (goweb.Configurer, error) {
		return goweb.ConfigurerFunc(func(_ context.Context, registry *goweb.Registry) error {
			registry.UseErrorMapper(goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
				if errors.Is(err, errStarterMapped) {
					return arkweb.Text(http.StatusConflict, "mapped")
				}
				return nil
			}))
			return nil
		}), nil
	}, container.WithFactoryDependencies("testErrorMapperDependency"))
}

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

type starterHTTPClientCustomizerConfiguration struct{}

func (starterHTTPClientCustomizerConfiguration) Name() string {
	return "test.web.http-client-customizer"
}

func (starterHTTPClientCustomizerConfiguration) Order() int {
	return 0
}

func (c starterHTTPClientCustomizerConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterHTTPClientCustomizerConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	if err := gbcweb.RegisterHTTPClientBuilderCustomizer(config.Registry(), "testFirstHTTPClientCustomizer", gbcweb.HTTPClientBuilderCustomizerFunc(func(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return builder.DefaultHeader("X-Chain", "first").DefaultCookieValue("phase", "first"), nil
	}), container.WithOrder(-100)); err != nil {
		return err
	}
	return gbcweb.RegisterHTTPClientBuilderCustomizer(config.Registry(), "testSecondHTTPClientCustomizer", gbcweb.HTTPClientBuilderCustomizerFunc(func(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return builder.DefaultHeader("X-Chain", "second").DefaultCookieValue("mode", "second"), nil
	}), container.WithOrder(100))
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

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q failed: %v", path, err)
	}
}

func clearConfigDataEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		configdata.EnvConfigLocation,
		configdata.EnvConfigAdditionalLocation,
		configdata.EnvConfigName,
		configdata.EnvProfilesActive,
	} {
		t.Setenv(name, "")
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

func failHTTPServer(errors chan<- error, writer http.ResponseWriter, format string, args ...any) {
	select {
	case errors <- fmt.Errorf(format, args...):
	default:
	}
	http.Error(writer, "server assertion failed", http.StatusInternalServerError)
}

func assertNoHTTPServerError(t *testing.T, errors <-chan error) {
	t.Helper()
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}
