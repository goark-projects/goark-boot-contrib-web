package gbcweb_test

import (
	"net/http"
	"strings"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type starterAdviceBodyInput struct {
	Name string `json:"name"`
}

func TestAutoConfigure_whenControllerAdviceMessageAdviceExists_shouldServeThroughArkhos(t *testing.T) {
	advice := mvc.NewRestControllerAdvice("test.mvc.message-advice").WithRequestBodyAdvice(
		goweb.RequestBodyAdviceFunc{
			After: func(_ *arkweb.Context, input goweb.RequestBodyAdviceContext) error {
				target := input.Target.(*starterAdviceBodyInput)
				target.Name += "-request-advised"
				return nil
			},
		},
	).WithResponseAdvice(goweb.ResponseAdviceFunc(func(ctx *arkweb.Context, result arkweb.Result) (arkweb.Result, error) {
		ctx.Response().Header().Set("X-Controller-Advice", "response-advised")
		return result, nil
	}))
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.message-advice", mvc.NewRestController("advice",
			mvc.POST("/advice/messages", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input starterAdviceBodyInput) (map[string]string, error) {
				return map[string]string{"name": input.Name}, nil
			})),
		)).WithControllerAdvices(advice)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/advice/messages", strings.NewReader(`{"name":"goark"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		return request, nil
	}, http.StatusCreated)
	if snapshot.header.Get("X-Controller-Advice") != "response-advised" {
		t.Fatalf("X-Controller-Advice = %q, want response-advised", snapshot.header.Get("X-Controller-Advice"))
	}
	if snapshot.body != `{"name":"goark-request-advised"}` {
		t.Fatalf("body = %q, want advised request body", snapshot.body)
	}
}
