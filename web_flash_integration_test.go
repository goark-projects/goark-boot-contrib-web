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
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenRedirectFlashAttributesExist_shouldCarryFlashOnce(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
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
