package gbcweb_test

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/web/mvc"
)

type starterBindingSearchCriteria struct {
	Name string `form:"name" json:"name" arkarta:"required"`
	Page int    `form:"page" json:"page"`
}

type starterModelAttributeBindingPayload struct {
	Valid        bool   `json:"valid"`
	BindingError bool   `json:"bindingError"`
	Field        string `json:"field"`
	Name         string `json:"name"`
	Page         int    `json:"page"`
}

type starterBindingUploadRequest struct {
	Title string                `form:"title" arkarta:"required"`
	File  servletmultipart.Part `multipart:"file"`
}

type starterMultipartBindingPayload struct {
	Valid    bool   `json:"valid"`
	Field    string `json:"field"`
	Filename string `json:"filename"`
}

func TestAutoConfigure_whenModelAttributeBindingResultExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-attribute-binding-result", mvc.NewRestController("bindingSearch",
			mvc.GET("/binding-result/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterModelAttributeBindingPayload, error) {
				input, result, err := mvc.ModelAttributeResult[starterBindingSearchCriteria](ctx)
				if err != nil {
					return starterModelAttributeBindingPayload{}, err
				}
				field, _ := result.FieldError("name")
				return starterModelAttributeBindingPayload{
					Valid:        result.Valid(),
					BindingError: result.BindingError() != nil,
					Field:        field.Path(),
					Name:         input.Name,
					Page:         input.Page,
				}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	bindingSnapshot := requestUntilStatusSnapshot(t, serverURL+"/binding-result/search?name=goark&page=bad", http.StatusOK)
	var bindingPayload starterModelAttributeBindingPayload
	if err := arkjson.Unmarshal(nil, []byte(bindingSnapshot.body), &bindingPayload); err != nil {
		t.Fatalf("binding result json invalid: %v", err)
	}
	if bindingPayload.Valid || !bindingPayload.BindingError || bindingPayload.Field != "" || bindingPayload.Name != "goark" || bindingPayload.Page != 0 {
		t.Fatalf("binding payload = %#v, want captured binding error", bindingPayload)
	}

	validationSnapshot := requestUntilStatusSnapshot(t, serverURL+"/binding-result/search?page=2", http.StatusOK)
	var validationPayload starterModelAttributeBindingPayload
	if err := arkjson.Unmarshal(nil, []byte(validationSnapshot.body), &validationPayload); err != nil {
		t.Fatalf("validation result json invalid: %v", err)
	}
	if validationPayload.Valid || validationPayload.BindingError || validationPayload.Field != "name" || validationPayload.Page != 2 {
		t.Fatalf("validation payload = %#v, want captured validation error", validationPayload)
	}
}

func TestAutoConfigure_whenMultipartBindingResultExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.multipart-binding-result", mvc.NewRestController("bindingUploads",
			mvc.POST("/binding-result/uploads", mvc.BindMultipartResult(http.StatusOK,
				func(_ *arkweb.Context, input starterBindingUploadRequest, result mvc.BindingResult) (starterMultipartBindingPayload, error) {
					field, _ := result.FieldError("title")
					return starterMultipartBindingPayload{
						Valid:    result.Valid(),
						Field:    field.Path(),
						Filename: input.File.SubmittedFileName(),
					}, nil
				},
			)),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterFileOnlyMultipartBody(t)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/binding-result/uploads", strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)

	var payload starterMultipartBindingPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("multipart binding json invalid: %v", err)
	}
	if payload.Valid || payload.Field != "title" || payload.Filename != "profile.txt" {
		t.Fatalf("multipart binding payload = %#v, want captured validation error", payload)
	}
}

func starterFileOnlyMultipartBody(t testing.TB) (string, string) {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "profile.txt")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := io.WriteString(part, "hello"); err != nil {
		t.Fatalf("write part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	return body.String(), writer.FormDataContentType()
}

type starterRejectingValidator struct{}

func (starterRejectingValidator) Validate(_ context.Context, _ any) (validation.Result, error) {
	return validation.NewResult(validation.NewViolation("name", "reserved", "名称不可用", nil)), nil
}

type starterValidatorRequest struct {
	Name string `json:"name"`
}

type starterBindingResultPayload struct {
	Valid bool   `json:"valid"`
	Path  string `json:"path"`
	Code  string `json:"code"`
	Name  string `json:"name"`
}

func TestAutoConfigure_whenValidatorExists_shouldBindMVCValidation(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterValidatorConfiguration{},
			mvc.NewConfiguration("test.mvc.validator", mvc.NewController("validator",
				mvc.POST("/validated", mvc.BindJSON(http.StatusCreated, func(_ *arkweb.Context, input starterValidatorRequest) (map[string]string, error) {
					return map[string]string{"name": input.Name}, nil
				})),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/validated", strings.NewReader(`{"name":"root"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		return request, nil
	}, http.StatusUnprocessableEntity)
	if !strings.Contains(snapshot.body, `"code":"reserved"`) || !strings.Contains(snapshot.body, `"path":"name"`) {
		t.Fatalf("validation body = %q, want custom violation", snapshot.body)
	}
}

func TestAutoConfigure_whenBindingResultRouteExists_shouldHandleValidationResult(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterValidatorConfiguration{},
			mvc.NewConfiguration("test.mvc.binding-result", mvc.NewRestController("bindingResult",
				mvc.POST("/binding-result", mvc.BindJSONResult(http.StatusOK,
					func(_ *arkweb.Context, input starterValidatorRequest, result mvc.BindingResult) (starterBindingResultPayload, error) {
						field, ok := result.FieldError("name")
						if !ok {
							return starterBindingResultPayload{Valid: result.Valid(), Name: input.Name}, nil
						}
						return starterBindingResultPayload{
							Valid: result.Valid(),
							Path:  field.Path(),
							Code:  field.Code(),
							Name:  input.Name,
						}, nil
					},
				)),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusWith(t, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, starterServerURL(t, app)+"/binding-result", strings.NewReader(`{"name":"root"}`))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", arkjson.ContentType)
		request.Header.Set("Accept", arkjson.ContentType)
		return request, nil
	}, http.StatusOK)

	var payload starterBindingResultPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("binding result json invalid: %v", err)
	}
	if payload.Valid || payload.Path != "name" || payload.Code != "reserved" || payload.Name != "root" {
		t.Fatalf("binding result payload = %#v, want controller-handled validation result", payload)
	}
}

type starterValidatorConfiguration struct{}

func (starterValidatorConfiguration) Name() string {
	return "test.web.validator"
}

func (starterValidatorConfiguration) Order() int {
	return 0
}

func (c starterValidatorConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterValidatorConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return gbcweb.RegisterValidator(config.Registry(), "testRejectingValidator", starterRejectingValidator{})
}
