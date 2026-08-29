package gbcweb_test

import (
	"net/http"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
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
