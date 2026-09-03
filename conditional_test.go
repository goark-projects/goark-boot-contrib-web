package gbcweb_test

import (
	"net/http"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenConditionalRequestMatches_shouldReturnNotModified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	modified := time.Date(2026, time.August, 29, 8, 30, 0, 900, time.FixedZone("CST", 8*60*60))

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.conditional", mvc.NewController("conditional",
			mvc.GET("/reports/summary", mvc.Handler(func(ctx *arkweb.Context) (arkweb.Result, error) {
				if goweb.CheckNotModified(ctx, "summary-v1", modified) {
					return nil, nil
				}
				return goweb.OK(map[string]string{"state": "fresh"}).
					WithETag("summary-v1").
					WithLastModified(modified), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	first := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/reports/summary", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)
	if first.header.Get("ETag") != `"summary-v1"` ||
		first.header.Get("Last-Modified") != "Sat, 29 Aug 2026 00:30:00 GMT" ||
		first.body != `{"state":"fresh"}` {
		t.Fatalf("initial response = %#v body %q", first.header, first.body)
	}

	second := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/reports/summary", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		request.Header.Set("If-None-Match", first.header.Get("ETag"))
		return request, nil
	}, http.StatusNotModified)
	if second.header.Get("ETag") != `"summary-v1"` ||
		second.header.Get("Last-Modified") != "Sat, 29 Aug 2026 00:30:00 GMT" ||
		second.body != "" {
		t.Fatalf("not modified response = %#v body %q", second.header, second.body)
	}
}
