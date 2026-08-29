package gbcweb_test

import (
	"net/http"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type starterRequestEntityInput struct {
	Name string `json:"name" arkarta:"required"`
}

type starterRequestEntityPayload struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Method        string `json:"method"`
	URL           string `json:"url"`
	Path          string `json:"path"`
	TraceID       string `json:"traceId"`
	ContentLength int64  `json:"contentLength"`
	HasBody       bool   `json:"hasBody"`
}

func TestAutoConfigure_whenRequestEntityRouteExists_shouldServeThroughArkhos(t *testing.T) {
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
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")))),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-entity", mvc.NewRestController("requestEntity",
			mvc.POST("/request-entity/{id}", mvc.BindRequestEntity(http.StatusAccepted,
				func(ctx *arkweb.Context, entity goweb.RequestEntity[starterRequestEntityInput]) (starterRequestEntityPayload, error) {
					id, err := mvc.PathString(ctx, "id")
					if err != nil {
						return starterRequestEntityPayload{}, err
					}
					traceID, _ := entity.HeaderValue("X-Trace-ID")
					body, hasBody := entity.Body()
					return starterRequestEntityPayload{
						ID:            id,
						Name:          body.Name,
						Method:        entity.Method(),
						URL:           entity.URL(),
						Path:          entity.Path(),
						TraceID:       traceID,
						ContentLength: entity.ContentLength(),
						HasBody:       hasBody && entity.HasBody(),
					}, nil
				})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	target := starterServerURL(t, app) + "/request-entity/42?mode=full"
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, target, strings.NewReader(`{"name":"goark"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		request.Header.Set("X-Trace-ID", "trace-1")
		return request, nil
	}, http.StatusAccepted)

	var payload starterRequestEntityPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response json invalid: %v", err)
	}
	if payload.ID != "42" || payload.Name != "goark" || payload.Method != http.MethodPost ||
		payload.URL != target || payload.Path != "/request-entity/42" || payload.TraceID != "trace-1" ||
		payload.ContentLength != int64(len(`{"name":"goark"}`)) || !payload.HasBody {
		t.Fatalf("request entity payload = %#v, want metadata and body", payload)
	}
}
