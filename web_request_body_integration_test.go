package gbcweb_test

import (
	"net/http"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterOptionalBodyInput struct {
	Name string `json:"name" arkarta:"required"`
}

type starterOptionalBodyPayload struct {
	Present bool   `json:"present"`
	Name    string `json:"name"`
}

func TestAutoConfigure_whenOptionalRequestBodyExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.optional-request-body", mvc.NewRestController("optionalBody",
			mvc.POST("/body/optional", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterOptionalBodyPayload, error) {
				input, present, err := mvc.OptionalValidatedRequestBody[starterOptionalBodyInput](ctx)
				if err != nil {
					return starterOptionalBodyPayload{}, err
				}
				return starterOptionalBodyPayload{Present: present, Name: input.Name}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	baseURL := starterServerURL(t, app)
	absent := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/body/optional", nil)
	}, http.StatusOK)
	if absent.body != `{"present":false,"name":""}` {
		t.Fatalf("absent body = %q, want optional body skipped", absent.body)
	}

	present := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/body/optional", strings.NewReader(`{"name":"goark"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)
	if present.body != `{"present":true,"name":"goark"}` {
		t.Fatalf("present body = %q, want optional body bound", present.body)
	}
}
