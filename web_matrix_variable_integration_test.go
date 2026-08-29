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

type starterMatrixVariablePayload struct {
	Owner string `json:"owner"`
	Pet   string `json:"pet"`
	Color string `json:"color"`
}

func TestAutoConfigure_whenMatrixVariablePathScopeExists_shouldServeThroughArkhos(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.matrix-variable", mvc.NewRestController("matrix",
			mvc.GET("/owners/{ownerId}/pets/{petId}", mvc.JSON(http.StatusOK, func(ctx *arkweb.Context) (starterMatrixVariablePayload, error) {
				owner, err := mvc.MatrixVariableString(ctx, "q", mvc.WithMatrixPathVariable("ownerId"))
				if err != nil {
					return starterMatrixVariablePayload{}, err
				}
				pet, err := mvc.MatrixVariableString(ctx, "q", mvc.WithMatrixPathVariable("petId"))
				if err != nil {
					return starterMatrixVariablePayload{}, err
				}
				color, err := mvc.MatrixVariableString(ctx, "color", mvc.WithMatrixPathVariable("petId"))
				if err != nil {
					return starterMatrixVariablePayload{}, err
				}
				return starterMatrixVariablePayload{Owner: owner, Pet: pet, Color: color}, nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/owners/42;q=owner/pets/21;q=pet;color=black", http.StatusOK)
	var got starterMatrixVariablePayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &got); err != nil {
		t.Fatalf("response JSON invalid: %v", err)
	}
	if got != (starterMatrixVariablePayload{Owner: "owner", Pet: "pet", Color: "black"}) {
		t.Fatalf("payload = %#v, want path-scoped matrix variables", got)
	}
}
