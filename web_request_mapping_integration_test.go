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

type starterPathPrefixPayload struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func TestAutoConfigure_whenRequestMappingWithoutMethodExists_shouldServeDefaultMethodsThroughArkhos(t *testing.T) {
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
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
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

	requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodTrace, serverURL+"/request-mapping", nil)
	}, http.StatusMethodNotAllowed)
}

func TestAutoConfigure_whenControllerPathPrefixesExist_shouldServePrefixedRoutesThroughArkhos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	controller := mvc.NewRestController("prefixed",
		mvc.GET("/users/{id}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterPathPrefixPayload, error) {
			id, err := mvc.PathString(ctx, "id")
			if err != nil {
				return starterPathPrefixPayload{}, err
			}
			return starterPathPrefixPayload{
				ID:   id,
				Path: ctx.Request().Path(),
			}, nil
		})),
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
