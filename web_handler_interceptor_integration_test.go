package gbcweb_test

import (
	"net/http"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	"goark.dev/gbc-web"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenMVCHandlerInterceptorsExist_shouldServeThroughArkhos(t *testing.T) {
	mapping, err := goweb.NewInterceptorMapping(
		goweb.WithInterceptorPathPatterns("/api/**"),
		goweb.WithInterceptorExcludePathPatterns("/api/public/**"),
	)
	if err != nil {
		t.Fatalf("NewInterceptorMapping failed: %v", err)
	}
	configuration := mvc.NewConfiguration("test.mvc.handler-interceptors", mvc.NewRestController("accounts",
		mvc.GET("/api/accounts", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "accounts", nil
		})),
		mvc.GET("/api/public/ping", mvc.Text(http.StatusOK, func(_ *arkweb.Context) (string, error) {
			return "pong", nil
		})),
	)).WithHandlerInterceptors(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Handler-Interceptor", "global")
			return true, nil
		},
	}).WithMappedHandlerInterceptor(mvc.HandlerInterceptorFuncs{
		PreHandleFunc: func(ctx *arkweb.Context) (bool, error) {
			ctx.Response().Header().Set("X-MVC-Mapped-Handler-Interceptor", "api")
			return true, nil
		},
	}, mapping)
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(configuration),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	matched := requestUntilStatusSnapshot(t, serverURL+"/api/accounts", http.StatusOK)
	if matched.header.Get("X-MVC-Handler-Interceptor") != "global" ||
		matched.header.Get("X-MVC-Mapped-Handler-Interceptor") != "api" {
		t.Fatalf("matched headers = %#v, want global and mapped interceptors", matched.header)
	}
	if matched.body != "accounts" {
		t.Fatalf("matched body = %q, want accounts", matched.body)
	}
	excluded := requestUntilStatusSnapshot(t, serverURL+"/api/public/ping", http.StatusOK)
	if excluded.header.Get("X-MVC-Handler-Interceptor") != "global" {
		t.Fatalf("excluded global header = %q, want global", excluded.header.Get("X-MVC-Handler-Interceptor"))
	}
	if got := excluded.header.Get("X-MVC-Mapped-Handler-Interceptor"); got != "" {
		t.Fatalf("excluded mapped header = %q, want empty", got)
	}
	if excluded.body != "pong" {
		t.Fatalf("excluded body = %q, want pong", excluded.body)
	}
}
