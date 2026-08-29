package gbcweb_test

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterRequestPartMetadata struct {
	Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" arkarta:"required"`
}

func TestAutoConfigure_whenJSONRequestPartExists_shouldBindValidateAndServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-part", mvc.NewRestController("parts",
			mvc.POST("/parts", mvc.JSON(http.StatusCreated, func(ctx *arkweb.Context) (map[string]string, error) {
				metadata, err := mvc.ValidatedRequestPartJSON[starterRequestPartMetadata](ctx, "metadata", []string{"create"})
				if err != nil {
					return nil, err
				}
				file, err := mvc.RequestPart(ctx, "file")
				if err != nil {
					return nil, err
				}
				reader, err := file.Open()
				if err != nil {
					return nil, err
				}
				defer reader.Close()
				body, err := io.ReadAll(reader)
				if err != nil {
					return nil, err
				}
				return map[string]string{
					"name":     metadata.Name,
					"filename": file.SubmittedFileName(),
					"body":     string(body),
				}, nil
			}), mvc.WithConsumes("multipart/form-data")),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	invalid := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterJSONRequestPartBody(t, `{}`, "application/vnd.goark.metadata+json")
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/parts", strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusUnprocessableEntity)
	if invalid.header.Get("Content-Type") != "application/problem+json" ||
		!strings.Contains(invalid.body, `"path":"name"`) {
		t.Fatalf("validation response = %d %#v %q, want name violation", http.StatusUnprocessableEntity, invalid.header, invalid.body)
	}

	unsupported := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterJSONRequestPartBody(t, `plain`, "text/plain")
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/parts", strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusUnsupportedMediaType)
	if unsupported.header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("unsupported Content-Type = %q, want problem json", unsupported.header.Get("Content-Type"))
	}

	created := requestUntilStatusWith(t, func() (*http.Request, error) {
		body, contentType := starterJSONRequestPartBody(t, `{"name":"avatar"}`, "application/vnd.goark.metadata+json")
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/parts", strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", contentType)
		return request, nil
	}, http.StatusCreated)
	var payload map[string]string
	if err := arkjson.Unmarshal(nil, []byte(created.body), &payload); err != nil {
		t.Fatalf("created JSON invalid: %v", err)
	}
	if payload["name"] != "avatar" || payload["filename"] != "profile.txt" || payload["body"] != "hello" {
		t.Fatalf("created payload = %#v", payload)
	}
}

func starterJSONRequestPartBody(t testing.TB, metadata string, metadataContentType string) (string, string) {
	t.Helper()
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "metadata",
		"filename": "metadata.json",
	}))
	header.Set("Content-Type", metadataContentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("CreatePart failed: %v", err)
	}
	if _, err := io.WriteString(part, metadata); err != nil {
		t.Fatalf("write metadata part failed: %v", err)
	}
	file, err := writer.CreateFormFile("file", "profile.txt")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := io.WriteString(file, "hello"); err != nil {
		t.Fatalf("write file part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close failed: %v", err)
	}
	return body.String(), writer.FormDataContentType()
}
