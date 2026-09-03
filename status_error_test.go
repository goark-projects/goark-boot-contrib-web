package gbcweb_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
	"goark.dev/goark/web/problem"
)

func TestAutoConfigure_whenHandlerReturnsStatusError_shouldUseProblemDetailsStatus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	cause := errors.New("internal quota bucket")

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.status-error", mvc.NewController("errors",
			mvc.GET("/limited", mvc.JSON(http.StatusOK, func(_ *arkweb.Context) (map[string]string, error) {
				return nil, goweb.NewStatusError(http.StatusTooManyRequests, "rate limited", cause)
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/limited", http.StatusTooManyRequests)
	if snapshot.header.Get("Content-Type") != problem.MediaType {
		t.Fatalf("Content-Type = %q, want problem json", snapshot.header.Get("Content-Type"))
	}
	if !strings.Contains(snapshot.body, `"status":429`) ||
		!strings.Contains(snapshot.body, `"detail":"rate limited"`) ||
		strings.Contains(snapshot.body, cause.Error()) {
		t.Fatalf("problem body = %q", snapshot.body)
	}
}
