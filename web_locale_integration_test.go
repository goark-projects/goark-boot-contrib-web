package gbcweb_test

import (
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type starterLocalePayload struct {
	OK         bool   `json:"ok"`
	Locale     string `json:"locale"`
	Language   string `json:"language"`
	Region     string `json:"region"`
	LocaleSize int    `json:"localeSize"`
}

func TestAutoConfigure_whenLocaleHelpersExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.locale", mvc.NewRestController("locale",
			mvc.GET("/locale", mvc.Entity(func(ctx *arkweb.Context) (goweb.ResponseEntity[starterLocalePayload], error) {
				locale, ok := goweb.RequestLocale(ctx)
				locales := goweb.RequestLocales(ctx)
				return goweb.OK(starterLocalePayload{
					OK:         ok,
					Locale:     locale.Tag(),
					Language:   locale.Language(),
					Region:     locale.Region(),
					LocaleSize: len(locales),
				}).WithContentLanguage(locale), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, serverURL+"/locale", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", arkjson.ContentType)
		request.Header.Set("Accept-Language", "en-US;q=0.8, zh-CN;q=0.9")
		return request, nil
	}, http.StatusOK)

	if snapshot.header.Get("Content-Language") != "zh-CN" {
		t.Fatalf("Content-Language = %q, want zh-CN", snapshot.header.Get("Content-Language"))
	}
	var payload starterLocalePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if !payload.OK || payload.Locale != "zh-CN" || payload.Language != "zh" || payload.Region != "CN" || payload.LocaleSize != 2 {
		t.Fatalf("payload = %#v, want locale details", payload)
	}
}
