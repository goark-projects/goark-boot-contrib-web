package gbcweb_test

import (
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterRequestMappingPayload struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func TestAutoConfigure_whenRequestMappingWithoutMethodExists_shouldServeDefaultMethodsThroughArkhos(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-mapping", mvc.NewRestController("requestMapping",
			mvc.RequestMapping("/request-mapping", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterRequestMappingPayload, error) {
				return starterRequestMappingPayload{
					Method: ctx.Request().Method(),
					Path:   ctx.Request().Path(),
				}, nil
			}))...,
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodTrace} {
		snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
			return http.NewRequestWithContext(t.Context(), method, serverURL+"/request-mapping", nil)
		}, http.StatusOK)
		var payload starterRequestMappingPayload
		if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
			t.Fatalf("%s response json invalid: %v", method, err)
		}
		if payload.Method != method || payload.Path != "/request-mapping" {
			t.Fatalf("%s payload = %#v, want method and path", method, payload)
		}
	}
}
