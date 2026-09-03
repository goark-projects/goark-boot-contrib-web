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

type starterRequestMethodPayload struct {
	Method string `json:"method"`
}

func TestAutoConfigure_whenControllerRequestMethodsExist_shouldServeCombinedMethodsThroughArkhos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	handler := mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterRequestMethodPayload, error) {
		return starterRequestMethodPayload{Method: ctx.Request().Method()}, nil
	})
	routes := mvc.RequestMapping("/controller-methods/implicit", handler)
	routes = append(routes, mvc.GET("/controller-methods/explicit", handler))
	controller := mvc.NewRestController("methods", routes...).
		WithRequestMethods(http.MethodPost, http.MethodTrace)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-methods", controller)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	for _, method := range []string{http.MethodPost, http.MethodTrace} {
		assertRequestMethodRoute(t, method, serverURL+"/controller-methods/implicit")
	}
	requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/controller-methods/implicit", nil)
	}, http.StatusMethodNotAllowed)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodTrace} {
		assertRequestMethodRoute(t, method, serverURL+"/controller-methods/explicit")
	}
}

func assertRequestMethodRoute(t *testing.T, method string, target string) {
	t.Helper()
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), method, target, nil)
	}, http.StatusOK)
	var payload starterRequestMethodPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("%s response json invalid: %v", method, err)
	}
	if payload.Method != method {
		t.Fatalf("payload method = %q, want %q", payload.Method, method)
	}
}
