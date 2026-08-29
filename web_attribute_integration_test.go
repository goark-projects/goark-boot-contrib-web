package gbcweb_test

import (
	"context"
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	"goark.dev/arkarta/servlet"
	"goark.dev/arkarta/servlet/session"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

type starterAttributeProfile struct {
	ID string `json:"id"`
}

type starterAttributePayload struct {
	TraceID   string `json:"traceId"`
	ProfileID string `json:"profileId"`
	Limit     int    `json:"limit"`
}

func TestAutoConfigure_whenTypedAttributesExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterAttributeConfiguration{},
			mvc.NewConfiguration("test.mvc.typed-attributes", mvc.NewRestController("typedAttributes",
				mvc.GET("/attributes/typed", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterAttributePayload, error) {
					profile, err := mvc.RequestAttribute[starterAttributeProfile](ctx, "profile")
					if err != nil {
						return starterAttributePayload{}, err
					}
					limit, err := mvc.RequestAttribute[int](ctx, "limit")
					if err != nil {
						return starterAttributePayload{}, err
					}
					traceID, err := mvc.SessionAttribute[string](ctx, "traceID")
					if err != nil {
						return starterAttributePayload{}, err
					}
					return starterAttributePayload{TraceID: traceID, ProfileID: profile.ID, Limit: limit}, nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/attributes/typed", http.StatusOK)
	var payload starterAttributePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if payload.TraceID != "trace-1" || payload.ProfileID != "p-1" || payload.Limit != 42 {
		t.Fatalf("payload = %#v, want typed attributes", payload)
	}
}

type starterAttributeConfiguration struct{}

func (starterAttributeConfiguration) Name() string {
	return "test.web.typed-attributes"
}

func (starterAttributeConfiguration) Order() int {
	return 0
}

func (c starterAttributeConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterAttributeConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return goweb.RegisterFilter(config.Registry(), "testTypedAttributeFilter", servlet.FilterFunc(func(ctx context.Context, req *servlet.Request, res servlet.Response, chain servlet.Chain) error {
		req.SetAttribute("profile", starterAttributeProfile{ID: "p-1"})
		req.SetAttribute("limit", "42")
		current, err := session.NewMemoryManager().Create(ctx)
		if err != nil {
			return err
		}
		if err := current.SetAttribute("traceID", "trace-1"); err != nil {
			return err
		}
		req.SetAttribute(session.AttributeCurrentSession, current)
		return chain.Next(ctx, req, res)
	}))
}
