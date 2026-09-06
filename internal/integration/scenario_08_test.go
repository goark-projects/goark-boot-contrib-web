package gbcweb_test

import (
	"context"
	"net/http"
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
	"goark.dev/goark"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenRedirectFlashAttributesExist_shouldCarryFlashOnce(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.web.flash.mvc", mvc.NewController(
			"flash",
			mvc.POST("/flash", mvc.Return(0, func(*arkweb.Context) (mvc.ModelAndView, error) {
				attributes := mvc.NewRedirectAttributes().
					AddAttribute("id", "42").
					AddFlashAttribute("notice", "created")
				return mvc.Redirect(
					"/flash/target",
					attributes,
					mvc.WithViewStatus(http.StatusSeeOther),
				), nil
			})),
			mvc.GET(
				"/flash/target",
				mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
					notice, err := mvc.FlashAttribute[string](
						ctx,
						"notice",
						mvc.WithRequired(false),
					)
					if err != nil {
						return nil, err
					}
					modelNotice, _ := mvc.CurrentModel(ctx).Attribute("notice")
					return map[string]string{
						"notice":      notice,
						"modelNotice": stringValue(modelNotice),
					}, nil
				}),
			),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	noRedirectClient := http.Client{
		Timeout: time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	first := requestUntilStatusWithClient(t, noRedirectClient, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/flash", nil)
	}, http.StatusSeeOther)
	if got := first.header.Get("Location"); got != "/flash/target?id=42" {
		t.Fatalf("Location = %q, want /flash/target?id=42", got)
	}
	sessionCookie := first.header.Get("Set-Cookie")
	if sessionCookie == "" {
		t.Fatal("missing flash session cookie")
	}
	cookieHeader := strings.SplitN(sessionCookie, ";", 2)[0]

	second := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/flash/target?id=42",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Cookie", cookieHeader)
		return request, nil
	}, http.StatusOK)
	var payload map[string]string
	if err := arkjson.Unmarshal(nil, []byte(second.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["notice"] != "created" || payload["modelNotice"] != "created" {
		t.Fatalf("payload = %#v, want flash values", payload)
	}

	third := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/flash/target?id=42",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Cookie", cookieHeader)
		return request, nil
	}, http.StatusOK)
	if err := arkjson.Unmarshal(nil, []byte(third.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["notice"] != "" || payload["modelNotice"] != "" {
		t.Fatalf("payload = %#v, want consumed flash values", payload)
	}
}

func TestAutoConfigure_whenModelAttributeSourcesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.model-attribute-sources", mvc.NewRestController(
				"modelAttributeSources",
				mvc.GET(
					"/tenants/{tenantId}/users/{userId}",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterModelAttributeSourcesCriteria, error) {
							return mvc.ModelAttribute[starterModelAttributeSourcesCriteria](ctx)
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

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			starterServerURL(
				t,
				app,
			)+"/tenants/core;scope=internal/users/42;role=admin?tenantId=query&mode=query",
			nil,
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("X-Request-Id", "req-1")
		request.Header.Set("Accept-Language", "zh-CN")
		request.Header.Set("Mode", "header")
		return request, nil
	}, http.StatusOK)

	var got starterModelAttributeSourcesCriteria
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got.TenantID != "query" ||
		got.UserID != 42 ||
		got.RequestID != "req-1" ||
		got.AcceptLanguage != "zh-CN" ||
		got.Mode != "query" {
		t.Fatalf(
			"criteria = %#v, want model attribute sources with request parameter priority",
			got,
		)
	}
}

func TestAutoConfigure_whenModelNameInferred_shouldRenderTemplateThroughArkhos(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates", "accounts")
	mkdir(t, templateDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(
		t,
		filepath.Join(templateDir, "detail.html"),
		"<h1>{{.starterModelNameAccount.Name}}</h1>",
	)
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(resource)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.model-name", mvc.NewController(
				"accounts",
				mvc.GET(
					"/accounts/detail",
					mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
						return mvc.NewModelAndView(
							"accounts/detail",
							starterModelNameAccount{Name: "Goark"},
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

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/accounts/detail",
		http.StatusOK,
	)
	if snapshot.body != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want rendered inferred model", snapshot.body)
	}
}

func TestAutoConfigure_whenDependentErrorMapperConfigurerExists_shouldApplyBeforeProblemDetails(
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
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.dependent-error", mvc.NewController(
				"dependent-errors",
				mvc.GET(
					"/dependent-errors",
					mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
						return nil, errStarterMapped
					}),
				),
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
func (starterWebSocketConfiguration) RegisterWithContext(
	_ context.Context,
	config goark.ConfigurationContext,
) error {
	return gbcweb.RegisterWebSocketEndpoint(
		config.Registry(),
		"testChatWebSocket",
		"/ws/chat",
		gbcweb.WebSocketEndpointFunc{
			Text: func(ctx context.Context, session gbcweb.WebSocketSession, text string) error {
				return session.SendText(ctx, "echo:"+text)
			},
		},
		gbcweb.WithWebSocketServletName("testChatSocket"),
		gbcweb.WithWebSocketSubprotocols("chat"),
	)
}
func assertStarterPathPrefixPayload(t *testing.T, body string, id string, path string) {
	t.Helper()
	var payload starterPathPrefixPayload
	if err := arkjson.Unmarshal(nil, []byte(body), &payload); err != nil {
		t.Fatalf("path prefix response json invalid: %v", err)
	}
	if payload.ID != id || payload.Path != path {
		t.Fatalf("path prefix payload = %#v, want id=%q path=%q", payload, id, path)
	}
}

type starterModelAttributeSourcesCriteria struct {
	TenantID       string `form:"tenantId"   json:"tenantId"`
	UserID         int64  `form:"userId"     json:"userId"`
	RequestID      string `form:"xRequestId" json:"requestId"`
	AcceptLanguage string `                  json:"acceptLanguage"`
	Mode           string `form:"mode"       json:"mode"`
}
type starterForwardedPayload struct {
	URL    string `json:"url"`
	Remote string `json:"remote"`
}

func (starterTokenConverter) CanRead(target any, mediaType string) bool {
	_, ok := target.(*starterTokenInput)
	return ok && strings.HasPrefix(mediaType, starterTokenMediaType)
}

type starterTenantID struct {
	value string
}

type starterAttributeProfile struct {
	ID string `json:"id"`
}

type starterErrorMapperConfiguration struct{}

type starterMessageConverterConfiguration struct{}
