package gbcweb_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
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
		boot.WithConfiguration(mvc.NewConfiguration("test.web.flash.mvc", mvc.NewController("flash",
			mvc.POST("/flash", mvc.Return(0, func(*arkweb.Context) (mvc.ModelAndView, error) {
				attributes := mvc.NewRedirectAttributes().
					AddAttribute("id", "42").
					AddFlashAttribute("notice", "created")
				return mvc.Redirect("/flash/target", attributes, mvc.WithViewStatus(http.StatusSeeOther)), nil
			})),
			mvc.GET("/flash/target", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (map[string]string, error) {
				notice, err := mvc.FlashAttribute[string](ctx, "notice", mvc.WithRequired(false))
				if err != nil {
					return nil, err
				}
				modelNotice, _ := mvc.CurrentModel(ctx).Attribute("notice")
				return map[string]string{
					"notice":      notice,
					"modelNotice": stringValue(modelNotice),
				}, nil
			})),
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
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/flash/target?id=42", nil)
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
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/flash/target?id=42", nil)
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

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type starterSessionAttributesPayload struct {
	Draft      string `json:"draft"`
	ModelDraft string `json:"modelDraft"`
}

func TestAutoConfigure_whenControllerSessionAttributesExist_shouldPersistUntilComplete(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.web.session-attributes", mvc.NewRestController("wizard",
			mvc.POST("/wizard/start", mvc.NoContent(func(ctx *arkweb.Context) error {
				mvc.CurrentModel(ctx).AddAttribute("draft", "step1")
				return nil
			})),
			mvc.GET("/wizard/current", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterSessionAttributesPayload, error) {
				draft, err := mvc.SessionAttribute[string](ctx, "draft", mvc.WithRequired(false))
				if err != nil {
					return starterSessionAttributesPayload{}, err
				}
				modelDraft, _ := mvc.CurrentModel(ctx).Attribute("draft")
				return starterSessionAttributesPayload{
					Draft:      draft,
					ModelDraft: sessionAttributeString(modelDraft),
				}, nil
			})),
			mvc.POST("/wizard/complete", mvc.NoContent(func(ctx *arkweb.Context) error {
				mvc.SetSessionComplete(ctx)
				return nil
			})),
		).WithSessionAttributes("draft"))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	started := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/wizard/start", nil)
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
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/wizard/complete", nil)
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

func requestStarterSessionAttributesPayload(t *testing.T, serverURL string, cookieHeader string) starterSessionAttributesPayload {
	t.Helper()

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/wizard/current", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Cookie", cookieHeader)
		return request, nil
	}, http.StatusOK)
	var payload starterSessionAttributesPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	return payload
}

func sessionAttributeString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func TestAutoConfigure_whenMVCHandlerInterceptorsExist_shouldServeThroughArkhos(t *testing.T) {
	mapping, err := goweb.NewInterceptorMapping(
		goweb.WithInterceptorPathPatterns("/api/**"),
		goweb.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	configuration := mvc.NewConfiguration("test.mvc.handler-interceptors", mvc.NewRestController("accounts",
		mvc.GET("/api/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "accounts", nil
		})),
		mvc.GET("/api/public/ping", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "pong", nil
		})),
	)).WithHandlerInterceptors(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Handler-Interceptor", "global")
			return true, nil
		},
	}).WithMappedHandlerInterceptor(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Mapped-Handler-Interceptor", "api")
			return true, nil
		},
	}, mapping)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(configuration),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	matched := requestUntilStatusSnapshot(t, serverURL+"/api/accounts", http.StatusOK)
	if matched.header.Get("X-MVC-Handler-Interceptor") != "global" ||
		matched.header.Get("X-MVC-Mapped-Handler-Interceptor") != "api" {
		t.Fatalf("matched headers = %#v, want global and mapped interceptors", matched.header)
	}
	if matched.body != "accounts" {
		t.Fatalf("matched body = %q, want accounts", matched.body)
	}
	excluded := requestUntilStatusSnapshot(t, serverURL+"/api/public/ping", http.StatusOK)
	if excluded.header.Get("X-MVC-Handler-Interceptor") != "global" {
		t.Fatalf("excluded global header = %q, want global", excluded.header.Get("X-MVC-Handler-Interceptor"))
	}
	if got := excluded.header.Get("X-MVC-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("excluded mapped header = %q, want empty", got)
	}
	if excluded.body != "pong" {
		t.Fatalf("excluded body = %q, want pong", excluded.body)
	}
}
