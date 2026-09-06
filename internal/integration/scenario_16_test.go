package gbcweb_test

import (
	"context"
	"errors"
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
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenResponseEntityRouteExists_shouldPreserveStatusHeadersAndBody(
	t *testing.T,
) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.entity", mvc.NewController(
			"jobs",
			mvc.POST(
				"/jobs",
				mvc.Entity(
					func(ctx *arkweb.Context) (goweb.ResponseEntity[map[string]string], error) {
						return goweb.CreatedFromCurrentRequest(
							ctx,
							"/{id}",
							map[string]string{"id": "42"},
							map[string]string{"state": "created"},
						)
					},
				),
			),
			mvc.GET(
				"/jobs/1",
				mvc.Entity(
					func(_ *arkweb.Context) (goweb.ResponseEntity[map[string]string], error) {
						return goweb.Status(http.StatusAccepted, map[string]string{"state": "queued"}).
								WithHeader("X-Starter-Entity", "true"),
							nil
					},
				),
			),
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
	server, err := goark.Get[*gbcarkhos.EmbeddedServer](
		t.Context(),
		appContext,
		gbcarkhos.BeanNameServer,
	)
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
	created := requestUntilStatusSnapshotWithMethod(
		t,
		http.MethodPost,
		server.URL()+"/jobs",
		http.StatusCreated,
	)
	if got := created.header.Get("Location"); got != server.URL()+"/jobs/42" {
		t.Fatalf("created Location = %q, want current request URI", got)
	}
	if created.body != `{"state":"created"}` {
		t.Fatalf("created body = %q, want entity json", created.body)
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
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.encoding", mvc.NewRestController(
				"encoding",
				mvc.POST(
					"/encoding/request",
					mvc.Return(http.StatusOK, func(ctx *arkweb.Context) (string, error) {
						return ctx.Request().CharacterEncoding(), nil
					}),
				),
				mvc.GET(
					"/encoding/response",
					mvc.Entity(func(_ *arkweb.Context) (goweb.ResponseEntity[string], error) {
						return goweb.Status(http.StatusOK, "ok").
								WithContentType("text/plain; charset=iso-8859-1"),
							nil
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
	requestSnapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/encoding/request",
			strings.NewReader("{}"),
		)
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
	if got := responseSnapshot.header.Get("Content-Type"); !strings.Contains(
		strings.ToLower(got),
		"charset=utf-8",
	) {
		t.Fatalf("response Content-Type = %q, want UTF-8 charset", got)
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
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.model-view", mvc.NewController(
				"pages",
				mvc.GET(
					"/reports/summary.html",
					mvc.Return(0, func(_ *arkweb.Context) (mvc.Model, error) {
						return mvc.NewModel().AddAttribute("Title", "Summary"), nil
					}),
				),
				mvc.GET(
					"/pages/42",
					mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
						model := mvc.NewModel().AddAttribute("Title", "Detail")
						return mvc.NewModelAndView(
							"pages/detail",
							model,
							mvc.WithViewStatus(http.StatusAccepted),
						), nil
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
	inferred := requestUntilStatusSnapshot(t, serverURL+"/reports/summary.html", http.StatusOK)
	if inferred.body != "<h1>Summary</h1>" {
		t.Fatalf("inferred model view body = %q, want summary", inferred.body)
	}
	explicit := requestUntilStatusSnapshot(t, serverURL+"/pages/42", http.StatusAccepted)
	if explicit.body != "<h1>Detail</h1>" {
		t.Fatalf("explicit model and view body = %q, want detail", explicit.body)
	}
}
func TestAutoConfigure_whenControllerPathPrefixesExist_shouldServePrefixedRoutesThroughArkhos(
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

	controller := mvc.NewRestController(
		"prefixed",
		mvc.GET(
			"/users/{id}",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterPathPrefixPayload, error) {
				id, err := mvc.PathString(ctx, "id")
				if err != nil {
					return starterPathPrefixPayload{}, err
				}
				return starterPathPrefixPayload{
					ID:   id,
					Path: ctx.Request().Path(),
				}, nil
			}),
		),
	).WithPathPrefixes("/api", "v2")
	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.path-prefixes", controller)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	api := requestUntilStatusSnapshot(t, serverURL+"/api/users/42", http.StatusOK)
	assertStarterPathPrefixPayload(t, api.body, "42", "/api/users/42")
	v2 := requestUntilStatusSnapshot(t, serverURL+"/v2/users/84", http.StatusOK)
	assertStarterPathPrefixPayload(t, v2.body, "84", "/v2/users/84")
	requestUntilStatusSnapshot(t, serverURL+"/users/42", http.StatusNotFound)
}

func (starterDependentErrorMapperConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	dependency := &starterErrorMapperDependency{}
	if err := container.RegisterInstance(
		config.Registry(), "testErrorMapperDependency", dependency,
	); err != nil {
		return err
	}
	return container.Register[goweb.Configurer](
		config.Registry(),
		"testDependentErrorMapper",
		func(_ context.Context, _ container.Resolver) (goweb.Configurer, error) {
			return goweb.ConfigurerFunc(func(_ context.Context, registry *goweb.Registry) error {
				registry.UseErrorMapper(
					goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
						if errors.Is(err, errStarterMapped) {
							return arkweb.Text(http.StatusConflict, "mapped")
						}
						return nil
					}),
				)
				return nil
			}), nil
		},
		container.WithFactoryDependencies("testErrorMapperDependency"),
	)
}

func (starterValidatorConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return gbcweb.RegisterValidator(
		config.Registry(),
		"testRejectingValidator",
		starterRejectingValidator{},
	)
}

func closeApp(t *testing.T, app *boot.Application) {
	t.Helper()
	if err := app.Close(t.Context()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}
}

type starterNestedSearchPayload struct {
	OwnerName  string `json:"ownerName"`
	OwnerLevel int    `json:"ownerLevel"`
	Page       int    `json:"page"`
}

type starterMappedSearchCriteria struct {
	Filters map[string]int      `form:"filters"`
	Tags    map[string][]string `form:"tags"`
}

func (starterDependentErrorMapperConfiguration) Order() int {
	return 0
}

func (starterTokenConverter) MediaTypes() []string {
	return []string{starterTokenMediaType}
}

type starterRequestEntityInput struct {
	Name string `json:"name" arkarta:"required"`
}

type starterWebFeaturesConfiguration struct{}
