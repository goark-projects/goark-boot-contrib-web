package gbcweb

import (
	"context"
	"testing"

	arkweb "goark.dev/arkarta/web"
	goarkcontainer "goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
)

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
	if err := RegisterMessageConverter(registry, "testMessageConverter", testMessageConverter{}, goarkcontainer.WithOrder(-100)); err != nil {
		t.Fatalf("RegisterMessageConverter failed: %v", err)
	}
	if err := RegisterRequestBodyAdvice(registry, "testRequestBodyAdvice", testRequestBodyAdvice{}, goarkcontainer.WithOrder(-50)); err != nil {
		t.Fatalf("RegisterRequestBodyAdvice failed: %v", err)
	}
	if err := registerMessageIO(registry); err != nil {
		t.Fatalf("registerMessageIO failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	writer, err := goarkcontainer.Get[message.Writer](context.Background(), runtimeContainer, BeanNameMessageWriter)
	if err != nil {
		t.Fatalf("resolve message writer failed: %v", err)
	}
	if _, ok := writer.Converters()[0].(testMessageConverter); !ok {
		t.Fatalf("first writer converter = %T, want testMessageConverter", writer.Converters()[0])
	}

	reader, err := goarkcontainer.Get[message.Reader](context.Background(), runtimeContainer, BeanNameMessageReader)
	if err != nil {
		t.Fatalf("resolve message reader failed: %v", err)
	}
	if _, ok := reader.ReadConverters()[0].(testMessageConverter); !ok {
		t.Fatalf("first reader converter = %T, want testMessageConverter", reader.ReadConverters()[0])
	}
	if _, ok := reader.ReadAdvices()[0].(testRequestBodyAdvice); !ok {
		t.Fatalf("first reader advice = %T, want testRequestBodyAdvice", reader.ReadAdvices()[0])
	}

	configurer, err := goarkcontainer.Get[goweb.Configurer](context.Background(), runtimeContainer, BeanNameMessageIOConfigurer)
	if err != nil {
		t.Fatalf("resolve message configurer failed: %v", err)
	}
	if err := configurer.ConfigureWeb(context.Background(), goweb.NewRegistry()); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
}
