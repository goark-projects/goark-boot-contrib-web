package gbcweb_test

import (
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterModelAttributeSourcesCriteria struct {
	TenantID       string `form:"tenantId" json:"tenantId"`
	UserID         int64  `form:"userId" json:"userId"`
	RequestID      string `form:"xRequestId" json:"requestId"`
	AcceptLanguage string `json:"acceptLanguage"`
	Mode           string `form:"mode" json:"mode"`
}

func TestAutoConfigure_whenModelAttributeSourcesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-attribute-sources", mvc.NewRestController("modelAttributeSources",
			mvc.GET("/tenants/{tenantId}/users/{userId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterModelAttributeSourcesCriteria, error) {
				return mvc.ModelAttribute[starterModelAttributeSourcesCriteria](ctx)
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, starterServerURL(t, app)+"/tenants/core;scope=internal/users/42;role=admin?tenantId=query&mode=query", nil)
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
		t.Fatalf("criteria = %#v, want model attribute sources with request parameter priority", got)
	}
}
