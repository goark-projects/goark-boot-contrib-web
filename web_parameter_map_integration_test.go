package gbcweb_test

import (
	"net/http"
	"reflect"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterParameterMapPayload struct {
	Params       map[string]string   `json:"params"`
	ParamValues  map[string][]string `json:"paramValues"`
	Headers      map[string]string   `json:"headers"`
	HeaderValues map[string][]string `json:"headerValues"`
}

func TestAutoConfigure_whenParameterMapsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.parameter-map", mvc.NewRestController("search",
			mvc.GET("/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterParameterMapPayload, error) {
				params, err := mvc.RequestParamMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
				paramValues, err := mvc.RequestParamValuesMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
				headers, err := mvc.RequestHeaderMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
				headerValues, err := mvc.RequestHeaderValuesMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
				return starterParameterMapPayload{
					Params:       params,
					ParamValues:  paramValues,
					Headers:      headers,
					HeaderValues: headerValues,
				}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, starterServerURL(t, app)+"/search?tag=query&tag=web&q=goark", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Add("X-Role", "admin")
		request.Header.Add("X-Role", "ops")
		request.Header.Set("X-Request-ID", "req-1")
		return request, nil
	}, http.StatusOK)

	var got starterParameterMapPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got.Params["tag"] != "query" || got.Params["q"] != "goark" {
		t.Fatalf("params = %#v", got.Params)
	}
	if !reflect.DeepEqual(got.ParamValues["tag"], []string{"query", "web"}) {
		t.Fatalf("param values = %#v", got.ParamValues)
	}
	if got.Headers["X-Role"] != "admin" || got.Headers["X-Request-Id"] != "req-1" {
		t.Fatalf("headers = %#v", got.Headers)
	}
	if !reflect.DeepEqual(got.HeaderValues["X-Role"], []string{"admin", "ops"}) {
		t.Fatalf("header values = %#v", got.HeaderValues)
	}
}
