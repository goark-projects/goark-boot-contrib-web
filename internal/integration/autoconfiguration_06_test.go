package gbcweb_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	arkweb "goark.dev/arkarta/web"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	webclient "goark.dev/goark/web/client"
)

func (starterDependentErrorMapperConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	if err := container.RegisterInstance(config.Registry(), "testErrorMapperDependency", &starterErrorMapperDependency{}); err != nil {
		return err
	}
	return container.Register[goweb.Configurer](config.Registry(), "testDependentErrorMapper", func(_ context.Context, _ container.Resolver) (goweb.Configurer, error) {
		return goweb.ConfigurerFunc(func(_ context.Context, registry *goweb.Registry) error {
			registry.UseErrorMapper(goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
				if errors.Is(err, errStarterMapped) {
					return arkweb.Text(http.StatusConflict, "mapped")
				}
				return nil
			}))
			return nil
		}), nil
	}, container.WithFactoryDependencies("testErrorMapperDependency"))
}

func (starterErrorMapperConfiguration) Name() string {
	return "test.web.error-mapper"
}

func (starterErrorMapperConfiguration) Order() int {
	return 0
}

func (c starterErrorMapperConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterErrorMapperConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return goweb.RegisterErrorMapper(config.Registry(), "testErrorMapper", goweb.ErrorMapperFunc(func(_ *arkweb.Context, err error) arkweb.Result {
		if errors.Is(err, errStarterMapped) {
			return arkweb.Text(http.StatusConflict, "mapped")
		}
		return nil
	}))
}

type starterHTTPClientCustomizerConfiguration struct{}

func (starterHTTPClientCustomizerConfiguration) Name() string {
	return "test.web.http-client-customizer"
}

func (starterHTTPClientCustomizerConfiguration) Order() int {
	return 0
}

func (c starterHTTPClientCustomizerConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterHTTPClientCustomizerConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	if err := gbcweb.RegisterHTTPClientBuilderCustomizer(config.Registry(), "testFirstHTTPClientCustomizer", gbcweb.HTTPClientBuilderCustomizerFunc(func(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return builder.DefaultHeader("X-Chain", "first").DefaultCookieValue("phase", "first"), nil
	}), container.WithOrder(-100)); err != nil {
		return err
	}
	return gbcweb.RegisterHTTPClientBuilderCustomizer(config.Registry(), "testSecondHTTPClientCustomizer", gbcweb.HTTPClientBuilderCustomizerFunc(func(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return builder.DefaultHeader("X-Chain", "second").DefaultCookieValue("mode", "second"), nil
	}), container.WithOrder(100))
}

func closeApp(t *testing.T, app *boot.Application) {
	t.Helper()
	if err := app.Close(t.Context()); err != nil {
		t.Fatalf("close app failed: %v", err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q failed: %v", path, err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q failed: %v", path, err)
	}
}

func clearConfigDataEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		configdata.EnvConfigLocation,
		configdata.EnvConfigAdditionalLocation,
		configdata.EnvConfigName,
		configdata.EnvProfilesActive,
	} {
		t.Setenv(name, "")
	}
}

func assertConfigurationCount(t *testing.T, descriptors []goark.ConfigurationDescriptor, name string, want int) {
	t.Helper()

	got := 0
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			got++
		}
	}
	if got != want {
		t.Fatalf("configuration %q count = %d, want %d", name, got, want)
	}
}

func failHTTPServer(errors chan<- error, writer http.ResponseWriter, format string, args ...any) {
	select {
	case errors <- fmt.Errorf(format, args...):
	default:
	}
	http.Error(writer, "server assertion failed", http.StatusInternalServerError)
}

func assertNoHTTPServerError(t *testing.T, errors <-chan error) {
	t.Helper()
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}
