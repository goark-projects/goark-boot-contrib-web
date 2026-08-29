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
	"goark.dev/goark/web/problem"
)

type starterRequestConditionPayload struct {
	Produces string `json:"produces"`
}

func TestAutoConfigure_whenControllerRequestConditionsExist_shouldServeThroughArkhos(t *testing.T) {
	const routeMediaType = "application/vnd.goark.condition+json"
	const acceptHeader = routeMediaType + ", " + problem.MediaType

	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
goark:
  web:
    server:
      address: 127.0.0.1:0
`)

	controller := mvc.NewRestController("conditions",
		mvc.POST("/conditions", mvc.JSON(http.StatusCreated, func(ctx *arkweb.Context) (starterRequestConditionPayload, error) {
			produces, _ := ctx.Request().Attribute(mvc.AttributeProducesMediaType)
			return starterRequestConditionPayload{Produces: produces.(string)}, nil
		}),
			mvc.WithConsumes(routeMediaType),
			mvc.WithProduces(routeMediaType),
			mvc.WithParams("mode=fast"),
			mvc.WithHeaders("X-Route=enabled"),
		),
	).
		WithConsumes("application/json").
		WithProduces("application/json").
		WithParams("tenant=admin").
		WithHeaders("X-Tenant=admin")

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-conditions", controller)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/conditions?tenant=admin&mode=fast", strings.NewReader("{}"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", routeMediaType)
		request.Header.Set("Accept", acceptHeader)
		request.Header.Set("X-Tenant", "admin")
		request.Header.Set("X-Route", "enabled")
		return request, nil
	}, http.StatusCreated)
	if got := snapshot.header.Get("Content-Type"); got != routeMediaType {
		t.Fatalf("Content-Type = %q, want route media type", got)
	}
	var payload starterRequestConditionPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.Produces != routeMediaType {
		t.Fatalf("payload = %#v, want route produces media type", payload)
	}

	requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/conditions?mode=fast", strings.NewReader("{}"))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", routeMediaType)
		request.Header.Set("Accept", acceptHeader)
		request.Header.Set("X-Tenant", "admin")
		request.Header.Set("X-Route", "enabled")
		return request, nil
	}, http.StatusBadRequest)
}

func TestAutoConfigure_whenSamePathRequestConditionsExist_shouldDispatchThroughArkhos(t *testing.T) {
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
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-condition-dispatch", mvc.NewRestController("conditionDispatch",
			mvc.GET("/condition-dispatch", mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
				return map[string]string{"mode": "fast"}, nil
			}), mvc.WithParams("mode=fast")),
			mvc.GET("/condition-dispatch", mvc.JSON(http.StatusOK, func(*arkweb.Context) (map[string]string, error) {
				return map[string]string{"mode": "default"}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	assertConditionDispatchPayload(t, serverURL+"/condition-dispatch?mode=fast", "fast")
	assertConditionDispatchPayload(t, serverURL+"/condition-dispatch", "default")
}

func assertConditionDispatchPayload(t *testing.T, target string, wantMode string) {
	t.Helper()
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	}, http.StatusOK)
	var payload map[string]string
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload["mode"] != wantMode {
		t.Fatalf("payload = %#v, want mode %q", payload, wantMode)
	}
}
