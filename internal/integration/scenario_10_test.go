package gbcweb_test

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goark.dev/arkarta/servlet"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/arkarta/websocket/frame"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenControllerSessionAttributesExist_shouldPersistUntilComplete(
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
			mvc.NewConfiguration("test.web.session-attributes", mvc.NewRestController(
				"wizard",
				mvc.POST("/wizard/start", mvc.NoContent(func(ctx *arkweb.Context) error {
					mvc.CurrentModel(ctx).AddAttribute("draft", "step1")
					return nil
				})),
				mvc.GET(
					"/wizard/current",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterSessionAttributesPayload, error) {
							draft, err := mvc.SessionAttribute[string](
								ctx,
								"draft",
								mvc.WithRequired(false),
							)
							if err != nil {
								return starterSessionAttributesPayload{}, err
							}
							modelDraft, _ := mvc.CurrentModel(ctx).Attribute("draft")
							return starterSessionAttributesPayload{
								Draft:      draft,
								ModelDraft: sessionAttributeString(modelDraft),
							}, nil
						},
					),
				),
				mvc.POST("/wizard/complete", mvc.NoContent(func(ctx *arkweb.Context) error {
					mvc.SetSessionComplete(ctx)
					return nil
				})),
			).WithSessionAttributes("draft")),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	started := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/wizard/start",
			nil,
		)
	}, http.StatusNoContent)
	sessionCookie := started.header.Get("Set-Cookie")
	if sessionCookie == "" {
		t.Fatal("missing session cookie")
	}
	cookieHeader := strings.SplitN(sessionCookie, ";", 2)[0]

	current := requestStarterSessionAttributesPayload(t, serverURL, cookieHeader)
	if current.Draft != "step1" || current.ModelDraft != "step1" {
		t.Fatalf("current payload = %#v, want persisted draft", current)
	}

	requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			serverURL+"/wizard/complete",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Cookie", cookieHeader)
		return request, nil
	}, http.StatusNoContent)

	cleared := requestStarterSessionAttributesPayload(t, serverURL, cookieHeader)
	if cleared.Draft != "" || cleared.ModelDraft != "" {
		t.Fatalf("cleared payload = %#v, want empty draft", cleared)
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
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.forward", mvc.NewController(
				"forwards",
				mvc.GET("/source", mvc.Return(0, func(_ *arkweb.Context) (string, error) {
					return "forward:/target?from=source", nil
				})),
				mvc.GET(
					"/target",
					mvc.ResponseBody(
						http.StatusAccepted,
						func(ctx *arkweb.Context) (string, error) {
							forwardURI, _ := ctx.Request().
								Attribute(servlet.AttributeForwardRequestURI)
							uri, ok := forwardURI.(string)
							if !ok {
								return "missing-forward-attribute", nil
							}
							return ctx.QueryValue("from") + ":" + uri, nil
						},
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
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/source",
			nil,
		)
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
func TestAutoConfigure_whenMVCModelAttributeInitializerExists_shouldRenderTemplateModel(
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

func TestAutoConfigure_whenWebSocketEndpointRegistered_shouldUpgradeAndEcho(t *testing.T) {
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
		boot.WithConfiguration(starterWebSocketConfiguration{}),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	conn, reader, headers := dialStarterWebSocket(t, starterServerURL(t, app)+"/ws/chat", "chat")
	defer conn.Close()

	if headers.Get("Sec-Websocket-Protocol") != "chat" {
		t.Fatalf("Sec-WebSocket-Protocol = %q, want chat", headers.Get("Sec-Websocket-Protocol"))
	}
	writeClientFrame(
		t,
		conn,
		frame.New(frame.OpText, []byte("hello"), frame.WithMask(frame.MaskKey{1, 2, 3, 4})),
	)
	echo := readServerFrame(t, reader)
	if echo.OpCode() != frame.OpText || string(echo.Payload()) != "echo:hello" {
		t.Fatalf("echo frame = %s/%q, want text echo", echo.OpCode(), string(echo.Payload()))
	}
	writeClientFrame(
		t,
		conn,
		frame.New(frame.OpClose, nil, frame.WithMask(frame.MaskKey{4, 3, 2, 1})),
	)
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
func requestUntilStatusWith(
	t *testing.T,
	build func() (*http.Request, error),
	statusCode int,
) responseSnapshot {
	t.Helper()
	return requestUntilStatusWithClient(t, http.Client{Timeout: time.Second}, build, statusCode)
}

type starterScopedSearchPayload struct {
	Page        int    `json:"page"`
	Tenant      string `json:"tenant"`
	ParamTenant string `json:"paramTenant"`
}

type starterSearchCriteria struct {
	Page   int             `form:"page"`
	Tenant starterTenantID `form:"tenant"`
}

type starterAdviceError struct {
	id string
}

func (starterConversionConfiguration) Order() int {
	return 0
}

type starterRequestMethodPayload struct {
	Method string `json:"method"`
}

type starterResponseAdviceConfiguration struct{}

func (starterDependentErrorMapperConfiguration) Name() string {
	return "test.web.dependent-error-mapper"
}
