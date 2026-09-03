package gbcweb_test

import (
	"net/http"
	"path/filepath"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
	"goark.dev/goark/web/mvc"
)

type starterModelNameAccount struct {
	Name string
}

func TestAutoConfigure_whenModelNameInferred_shouldRenderTemplateThroughArkhos(t *testing.T) {
	root := t.TempDir()
	resource := filepath.Join(root, "resource")
	templateDir := filepath.Join(resource, "templates", "accounts")
	mkdir(t, templateDir)
	writeFile(t, filepath.Join(resource, "app.yml"), `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)
	writeFile(t, filepath.Join(templateDir, "detail.html"), "<h1>{{.starterModelNameAccount.Name}}</h1>")
	t.Chdir(root)
	clearConfigDataEnvironment(t)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(resource)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.model-name", mvc.NewController("accounts",
			mvc.GET("/accounts/detail", mvc.Return(0, func(_ *arkweb.Context) (mvc.ModelAndView, error) {
				return mvc.NewModelAndView("accounts/detail", starterModelNameAccount{Name: "Goark"}), nil
			})),
		))),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/accounts/detail", http.StatusOK)
	if snapshot.body != "<h1>Goark</h1>" {
		t.Fatalf("body = %q, want rendered inferred model", snapshot.body)
	}
}
