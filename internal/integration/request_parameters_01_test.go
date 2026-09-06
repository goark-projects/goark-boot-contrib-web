package gbcweb_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
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
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
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

type starterParameterMapPayload struct {
	Path         map[string]string   `json:"path"`
	Params       map[string]string   `json:"params"`
	ParamValues  map[string][]string `json:"paramValues"`
	Headers      map[string]string   `json:"headers"`
	HeaderValues map[string][]string `json:"headerValues"`
	Cookies      map[string]string   `json:"cookies"`
	CookieValues map[string][]string `json:"cookieValues"`
}

func TestAutoConfigure_whenParameterMapsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.parameter-map", mvc.NewRestController("search",
			mvc.GET("/tenants/{tenantId}/search/{userId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterParameterMapPayload, error) {
				path, err := mvc.PathVariableMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
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
				cookies, err := mvc.CookieValueMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
				cookieValues, err := mvc.CookieValueValuesMap(ctx)
				if err != nil {
					return starterParameterMapPayload{}, err
				}
				return starterParameterMapPayload{
					Path:         path,
					Params:       params,
					ParamValues:  paramValues,
					Headers:      headers,
					HeaderValues: headerValues,
					Cookies:      cookies,
					CookieValues: cookieValues,
				}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, starterServerURL(t, app)+"/tenants/core;scope=internal/search/42;role=admin?tag=query&tag=web&q=goark", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Add("X-Role", "admin")
		request.Header.Add("X-Role", "ops")
		request.Header.Set("X-Request-ID", "req-1")
		request.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
		request.AddCookie(&http.Cookie{Name: "role", Value: "admin"})
		request.AddCookie(&http.Cookie{Name: "role", Value: "ops"})
		return request, nil
	}, http.StatusOK)

	var got starterParameterMapPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !reflect.DeepEqual(got.Path, map[string]string{"tenantId": "core", "userId": "42"}) {
		t.Fatalf("path = %#v", got.Path)
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
	if !reflect.DeepEqual(got.Cookies, map[string]string{"theme": "dark", "role": "admin"}) {
		t.Fatalf("cookies = %#v", got.Cookies)
	}
	if !reflect.DeepEqual(got.CookieValues["role"], []string{"admin", "ops"}) ||
		!reflect.DeepEqual(got.CookieValues["theme"], []string{"dark"}) {
		t.Fatalf("cookie values = %#v", got.CookieValues)
	}
}

type starterRequestParamPayload struct {
	Query string   `json:"query"`
	Tags  []string `json:"tags"`
	IDs   []int64  `json:"ids"`
}

func TestAutoConfigure_whenEmptyArrayRequestParamsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.empty-array-request-param", mvc.NewRestController("requestParams",
			mvc.GET("/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterRequestParamPayload, error) {
				query, err := mvc.RequestParamString(ctx, "q")
				if err != nil {
					return starterRequestParamPayload{}, err
				}
				tags, err := mvc.RequestParamStrings(ctx, "tag")
				if err != nil {
					return starterRequestParamPayload{}, err
				}
				ids, err := mvc.RequestParamInt64s(ctx, "id")
				if err != nil {
					return starterRequestParamPayload{}, err
				}
				return starterRequestParamPayload{Query: query, Tags: tags, IDs: ids}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search?q[]=goark&tag[]=web&tag[]=mvc&id[]=1&id[]=2", http.StatusOK)
	var got starterRequestParamPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got.Query != "goark" ||
		!reflect.DeepEqual(got.Tags, []string{"web", "mvc"}) ||
		!reflect.DeepEqual(got.IDs, []int64{1, 2}) {
		t.Fatalf("payload = %#v, want empty array index request params", got)
	}
}
