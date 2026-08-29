package gbcweb_test

import (
	"net/http"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterSessionAttributesPayload struct {
	Draft      string `json:"draft"`
	ModelDraft string `json:"modelDraft"`
}

func TestAutoConfigure_whenControllerSessionAttributesExist_shouldPersistUntilComplete(t *testing.T) {
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
