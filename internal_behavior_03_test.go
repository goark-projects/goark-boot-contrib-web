package gbcweb

import (
	"context"
	"testing"

	"goark.dev/arkarta/validation"
	arkweb "goark.dev/arkarta/web"
	goarkcontainer "goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	coreenv "goark.dev/goark/core/env"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
)

func TestStaticResourceURLPrefix(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{pattern: "/static/*", want: "/static"},
		{pattern: "/assets/", want: "/assets"},
		{pattern: "/", want: ""},
		{pattern: "*.css", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := staticResourceURLPrefix(tt.pattern); got != tt.want {
				t.Fatalf("prefix = %q, want %q", got, tt.want)
			}
		})
	}
}

func newTestEnvironment(t *testing.T, values map[string]any) coreenv.Environment {
	t.Helper()

	environment, err := coreenv.NewStandardEnvironment()
	if err != nil {
		t.Fatalf("new environment failed: %v", err)
	}
	source, err := coreenv.NewMapPropertySource("test", values)
	if err != nil {
		t.Fatalf("new property source failed: %v", err)
	}
	if err := environment.PropertySources().AddFirst(source); err != nil {
		t.Fatalf("add property source failed: %v", err)
	}
	return environment
}

func TestRegisterMVCConversion_whenConvertersExist_shouldAssembleService(t *testing.T) {
	registry := goarkcontainer.NewRegistry()
	if err := RegisterConverter(
		registry, "testStringIntConverter",
		convert.ConverterFunc[string, int](func(value string) (int, error) {
			return len(value) + 10, nil
		})); err != nil {
		t.Fatalf("RegisterConverter failed: %v", err)
	}
	if err := registerMVCConversion(registry); err != nil {
		t.Fatalf("registerMVCConversion failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	service, err := goarkcontainer.Get[*convert.Service](
		context.Background(),
		runtimeContainer,
		BeanNameConversionService,
	)
	if err != nil {
		t.Fatalf("resolve conversion service failed: %v", err)
	}
	converted, err := convert.Convert[int](service, "abcd")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if converted != 14 {
		t.Fatalf("converted = %d, want 14", converted)
	}

	configurer, err := goarkcontainer.Get[goweb.Configurer](
		context.Background(),
		runtimeContainer,
		BeanNameConversionConfigurer,
	)
	if err != nil {
		t.Fatalf("resolve conversion configurer failed: %v", err)
	}
	if err := configurer.ConfigureWeb(context.Background(), goweb.NewRegistry()); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
}

type testMessageConverter struct{}

type testRequestBodyAdvice struct{}

func (testMessageConverter) MediaTypes() []string {
	return []string{"application/vnd.goark.test"}
}

func (testMessageConverter) CanRead(any, string) bool {
	return false
}

func (testMessageConverter) Read(*arkweb.Context, any, string) error {
	return nil
}

func (testMessageConverter) CanWrite(any, string) bool {
	return false
}

func (testMessageConverter) Write(*arkweb.Context, any, string) error {
	return nil
}

func (testRequestBodyAdvice) BeforeRead(*arkweb.Context, message.ReadAdviceContext) error {
	return nil
}

func (testRequestBodyAdvice) AfterRead(*arkweb.Context, message.ReadAdviceContext) error {
	return nil
}

func TestRegisterMessageIO_whenConvertersExist_shouldAssembleReaderWriter(t *testing.T) {
	registry := goarkcontainer.NewRegistry()
	if err := RegisterMessageConverter(
		registry, "testMessageConverter", testMessageConverter{},
		goarkcontainer.WithOrder(-100),
	); err != nil {
		t.Fatalf("RegisterMessageConverter failed: %v", err)
	}
	if err := RegisterRequestBodyAdvice(
		registry, "testRequestBodyAdvice", testRequestBodyAdvice{},
		goarkcontainer.WithOrder(-50),
	); err != nil {
		t.Fatalf("RegisterRequestBodyAdvice failed: %v", err)
	}
	if err := registerMessageIO(registry); err != nil {
		t.Fatalf("registerMessageIO failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	writer, err := goarkcontainer.Get[message.Writer](
		context.Background(),
		runtimeContainer,
		BeanNameMessageWriter,
	)
	if err != nil {
		t.Fatalf("resolve message writer failed: %v", err)
	}
	if _, ok := writer.Converters()[0].(testMessageConverter); !ok {
		t.Fatalf("first writer converter = %T, want testMessageConverter", writer.Converters()[0])
	}

	reader, err := goarkcontainer.Get[message.Reader](
		context.Background(),
		runtimeContainer,
		BeanNameMessageReader,
	)
	if err != nil {
		t.Fatalf("resolve message reader failed: %v", err)
	}
	if _, ok := reader.ReadConverters()[0].(testMessageConverter); !ok {
		t.Fatalf(
			"first reader converter = %T, want testMessageConverter",
			reader.ReadConverters()[0],
		)
	}
	if _, ok := reader.ReadAdvices()[0].(testRequestBodyAdvice); !ok {
		t.Fatalf("first reader advice = %T, want testRequestBodyAdvice", reader.ReadAdvices()[0])
	}

	configurer, err := goarkcontainer.Get[goweb.Configurer](
		context.Background(),
		runtimeContainer,
		BeanNameMessageIOConfigurer,
	)
	if err != nil {
		t.Fatalf("resolve message configurer failed: %v", err)
	}
	if err := configurer.ConfigureWeb(context.Background(), goweb.NewRegistry()); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
}

type testRejectingValidator struct{}

func (testRejectingValidator) Validate(_ context.Context, _ any) (validation.Result, error) {
	return validation.NewResult(validation.NewViolation("name", "reserved", "名称不可用", nil)), nil
}

func TestRegisterWebValidator_whenNoCustomValidator_shouldRegisterDefaultConfigurer(t *testing.T) {
	registry := goarkcontainer.NewRegistry()
	if err := registerWebValidator(registry); err != nil {
		t.Fatalf("registerWebValidator failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	validator, err := goarkcontainer.Get[validation.Validator](
		context.Background(),
		runtimeContainer,
		BeanNameValidator,
	)
	if err != nil {
		t.Fatalf("resolve default validator failed: %v", err)
	}
	if validator == nil {
		t.Fatal("validator is nil")
	}
	configurer, err := goarkcontainer.Get[goweb.Configurer](
		context.Background(),
		runtimeContainer,
		BeanNameValidatorConfigurer,
	)
	if err != nil {
		t.Fatalf("resolve validator configurer failed: %v", err)
	}
	webRegistry := goweb.NewRegistry()
	if err := configurer.ConfigureWeb(context.Background(), webRegistry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	if webRegistry.Validator() == nil {
		t.Fatal("web registry validator is nil")
	}
}

func TestRegisterWebValidator_whenCustomValidatorExists_shouldSelectCustomValidator(t *testing.T) {
	registry := goarkcontainer.NewRegistry()
	if err := RegisterValidator(
		registry, "testRejectingValidator", testRejectingValidator{},
	); err != nil {
		t.Fatalf("RegisterValidator failed: %v", err)
	}
	if err := registerWebValidator(registry); err != nil {
		t.Fatalf("registerWebValidator failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	validator, err := goarkcontainer.GetByType[validation.Validator](
		context.Background(),
		runtimeContainer,
	)
	if err != nil {
		t.Fatalf("resolve validator by type failed: %v", err)
	}
	if _, ok := validator.(testRejectingValidator); !ok {
		t.Fatalf("validator = %T, want testRejectingValidator", validator)
	}
	configurer, err := goarkcontainer.Get[goweb.Configurer](
		context.Background(),
		runtimeContainer,
		BeanNameValidatorConfigurer,
	)
	if err != nil {
		t.Fatalf("resolve validator configurer failed: %v", err)
	}
	webRegistry := goweb.NewRegistry()
	if err := configurer.ConfigureWeb(context.Background(), webRegistry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	if _, ok := webRegistry.Validator().(testRejectingValidator); !ok {
		t.Fatalf(
			"web registry validator = %T, want testRejectingValidator",
			webRegistry.Validator(),
		)
	}
}
