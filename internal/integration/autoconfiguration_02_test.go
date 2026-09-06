package gbcweb_test

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

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
				forwardURI, _ := ctx.Request().Attribute(servlet.AttributeForwardRequestURI)
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
