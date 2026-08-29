package gbcweb_test

import (
	"net/http"
	"reflect"
	"testing"

	arkjson "goark.dev/arkarta/json"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterRequestParamPayload struct {
	Query string   `json:"query"`
	Tags  []string `json:"tags"`
	IDs   []int64  `json:"ids"`
}

func TestAutoConfigure_whenEmptyArrayRequestParamsExist_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.empty-array-request-param", mvc.NewRestController("requestParams",
			mvc.GET("/search", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterRequestParamPayload, error) {
				query, err := mvc.RequestParamString(ctx, "q")
				if err != nil {
					return starterRequestParamPayload{}, err
				}
				tags, err := mvc.RequestParamStrings(ctx, "tag")
				if err != nil {
					return starterRequestParamPayload{}, err
				}
				ids, err := mvc.RequestParamInt64s(ctx, "id")
				if err != nil {
					return starterRequestParamPayload{}, err
				}
				return starterRequestParamPayload{Query: query, Tags: tags, IDs: ids}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search?q[]=goark&tag[]=web&tag[]=mvc&id[]=1&id[]=2", http.StatusOK)
	var got starterRequestParamPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got.Query != "goark" ||
		!reflect.DeepEqual(got.Tags, []string{"web", "mvc"}) ||
		!reflect.DeepEqual(got.IDs, []int64{1, 2}) {
		t.Fatalf("payload = %#v, want empty array index request params", got)
	}
}
