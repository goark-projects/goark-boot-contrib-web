package gbcweb_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	arkws "goark.dev/arkarta/websocket"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	webclient "goark.dev/goark/web/client"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenHTTPClientConfigured_shouldRegisterBuilderAndClient(t *testing.T) {
	serverErrors := make(chan error, 2)
	api := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Header.Get("X-App") != "boot" {
				failHTTPServer(
					serverErrors,
					writer,
					"X-App = %q, want boot",
					request.Header.Get("X-App"),
				)
				return
			}
			switch request.URL.Path {
			case "/api/ping":
				session, err := request.Cookie("sid")
				if err != nil || session.Value != "abc" {
					failHTTPServer(serverErrors, writer, "sid cookie = %#v err %v", session, err)
					return
				}
				_, _ = io.WriteString(writer, `{"status":"UP"}`)
			case "/api/builder":
				if request.Header.Get("X-Builder") != "yes" {
					failHTTPServer(
						serverErrors,
						writer,
						"X-Builder = %q, want yes",
						request.Header.Get("X-Builder"),
					)
					return
				}
				_, _ = io.WriteString(writer, "builder")
			case "/api/upload":
				if err := request.ParseMultipartForm(1 << 20); err != nil {
					failHTTPServer(serverErrors, writer, "parse multipart failed: %v", err)
					return
				}
				file, header, err := request.FormFile("file")
				if err != nil {
					failHTTPServer(serverErrors, writer, "form file failed: %v", err)
					return
				}
				defer file.Close()
				body, err := io.ReadAll(file)
				if err != nil {
					failHTTPServer(serverErrors, writer, "read file failed: %v", err)
					return
				}
				if request.FormValue("title") != "avatar" || header.Filename != "profile.txt" ||
					string(body) != "hello" {
					failHTTPServer(
						serverErrors,
						writer,
						"multipart = %q %q %q",
						request.FormValue("title"),
						header.Filename,
						string(body),
					)
					return
				}
				writer.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(writer, "uploaded")
			default:
				failHTTPServer(serverErrors, writer, "path = %q", request.URL.Path)
			}
		}),
	)
	defer api.Close()

	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
			gbcweb.WithHTTPClientBaseURL(api.URL+"/api"),
			gbcweb.WithHTTPClientTimeout(2*time.Second),
			gbcweb.WithHTTPClientMaxResponseBytes(64),
			gbcweb.WithHTTPClientDefaultHeader("X-App", "boot"),
		)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	appContext, ok := app.Context()
	if !ok {
		t.Fatal("expected application context")
	}
	defaultClient, err := goark.Get[*webclient.Client](
		t.Context(),
		appContext,
		gbcweb.BeanNameHTTPClient,
	)
	if err != nil {
		t.Fatalf("resolve http client failed: %v", err)
	}
	response, err := defaultClient.Get(
		t.Context(),
		"/ping",
		webclient.WithCookieValue("sid", "abc"),
	)
	if err != nil {
		t.Fatalf("default client get failed: %v", err)
	}
	var payload map[string]string
	if err := response.DecodeJSON(&payload); err != nil {
		t.Fatalf("decode client response failed: %v", err)
	}
	if payload["status"] != "UP" {
		t.Fatalf("payload = %#v", payload)
	}

	builder, err := goark.Get[*webclient.Builder](
		t.Context(),
		appContext,
		gbcweb.BeanNameHTTPClientBuilder,
	)
	if err != nil {
		t.Fatalf("resolve http client builder failed: %v", err)
	}
	derivedClient, err := builder.DefaultHeader("X-Builder", "yes").Build()
	if err != nil {
		t.Fatalf("derived client build failed: %v", err)
	}
	derivedResponse, err := derivedClient.Get(t.Context(), "/builder")
	if err != nil {
		t.Fatalf("derived client get failed: %v", err)
	}
	if derivedResponse.BodyString() != "builder" {
		t.Fatalf("derived body = %q, want builder", derivedResponse.BodyString())
	}
	uploadResponse, err := defaultClient.Post(t.Context(), "/upload",
		webclient.WithMultipartFields(map[string]string{"title": "avatar"}, webclient.MultipartFile{
			FieldName:   "file",
			FileName:    "profile.txt",
			ContentType: "text/plain",
			Body:        strings.NewReader("hello"),
		}),
	)
	if err != nil {
		t.Fatalf("multipart client post failed: %v", err)
	}
	if uploadResponse.StatusCode() != http.StatusCreated ||
		uploadResponse.BodyString() != "uploaded" {
		t.Fatalf(
			"upload response = %d %q, want 201 uploaded",
			uploadResponse.StatusCode(),
			uploadResponse.BodyString(),
		)
	}
	assertNoHTTPServerError(t, serverErrors)
}

func TestAutoConfigure_whenEmptyArrayIndexFieldsExist_shouldBindModelAttributesThroughArkhos(
	t *testing.T,
) {
	controller := mvc.NewRestController(
		"starter-empty-array-index",
		mvc.GET(
			"/field-array-indices",
			mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterArrayPayload, error) {
				input, err := mvc.ModelAttribute[starterArrayInput](ctx)
				if err != nil {
					return starterArrayPayload{}, err
				}
				out := starterArrayPayload{Tags: input.Tags}
				if input.Profile != nil {
					out.Aliases = input.Profile.Aliases
				}
				return out, nil
			}),
		),
	)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			mvc.NewConfiguration("test.mvc.empty-array-index", controller),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(
			t,
			app,
		)+"/field-array-indices?tags[]=red&tags[]=blue&profile.aliases[]=core&profile.aliases[]=web",
		http.StatusOK,
	)
	var payload starterArrayPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("empty array index json invalid: %v", err)
	}
	if len(payload.Tags) != 2 ||
		payload.Tags[0] != "red" ||
		payload.Tags[1] != "blue" ||
		len(payload.Aliases) != 2 ||
		payload.Aliases[0] != "core" ||
		payload.Aliases[1] != "web" {
		t.Fatalf("payload = %#v, want empty array index fields", payload)
	}
}

func TestAutoConfigure_whenValidatorExists_shouldBindMVCValidation(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterValidatorConfiguration{},
			mvc.NewConfiguration("test.mvc.validator", mvc.NewController(
				"validator",
				mvc.POST(
					"/validated",
					mvc.BindJSON(
						http.StatusCreated,
						func(_ *arkweb.Context, input starterValidatorRequest) (map[string]string, error) {
							return map[string]string{"name": input.Name}, nil
						},
					),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodPost,
			starterServerURL(t, app)+"/validated",
			strings.NewReader(`{"name":"root"}`),
		)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		return request, nil
	}, http.StatusUnprocessableEntity)
	if !strings.Contains(snapshot.body, `"code":"reserved"`) ||
		!strings.Contains(snapshot.body, `"path":"name"`) {
		t.Fatalf("validation body = %q, want custom violation", snapshot.body)
	}
}

func starterPreferencePayloadFromInput(input starterPreferenceInput) starterPreferencePayload {
	out := starterPreferencePayload{
		Theme:      input.Theme,
		TagsNil:    input.Tags == nil,
		TagsLength: len(input.Tags),
	}
	if input.Notify != nil {
		out.NotifySet = true
		out.Notify = *input.Notify
	}
	if input.Confirm != nil {
		out.ConfirmSet = true
		out.Confirm = *input.Confirm
	}
	if input.Profile != nil {
		out.ProfileSet = true
		out.Subscribed = input.Profile.Subscribed
	}
	return out
}

func assertRequestMethodRoute(t *testing.T, method string, target string) {
	t.Helper()
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(t.Context(), method, target, nil)
	}, http.StatusOK)
	var payload starterRequestMethodPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("%s response json invalid: %v", method, err)
	}
	if payload.Method != method {
		t.Fatalf("payload method = %q, want %q", payload.Method, method)
	}
}

func TestRegisterWebSocketEndpoint_whenEndpointIsNil_shouldReturnError(t *testing.T) {
	registry := container.NewRegistry()
	err := gbcweb.RegisterWebSocketEndpoint(registry, "badSocket", "/ws/bad", nil)
	if !errors.Is(err, arkws.ErrNilEndpoint) {
		t.Fatalf("err = %v, want ErrNilEndpoint", err)
	}
}

func (c starterDependentErrorMapperConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type responseSnapshot struct {
	body   string
	header http.Header
}

type starterSessionAttributesPayload struct {
	Draft      string `json:"draft"`
	ModelDraft string `json:"modelDraft"`
}

func (starterRequestBodyAdviceConfiguration) Order() int {
	return 0
}

func (starterWebFeaturesConfiguration) Name() string {
	return "test.web.features"
}
