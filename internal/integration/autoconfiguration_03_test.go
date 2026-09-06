package gbcweb_test

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

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
