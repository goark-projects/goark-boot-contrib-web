package gbcweb

import (
	"context"
	"testing"

	goarkcontainer "goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	goweb "goark.dev/goark/web"
)

func TestRegisterMVCConversion_whenConvertersExist_shouldAssembleService(t *testing.T) {
	registry := goarkcontainer.NewRegistry()
	if err := RegisterConverter(registry, "testStringIntConverter", convert.ConverterFunc[string, int](func(value string) (int, error) {
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

	service, err := goarkcontainer.Get[*convert.Service](context.Background(), runtimeContainer, BeanNameConversionService)
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

	configurer, err := goarkcontainer.Get[goweb.Configurer](context.Background(), runtimeContainer, BeanNameConversionConfigurer)
	if err != nil {
		t.Fatalf("resolve conversion configurer failed: %v", err)
	}
	if err := configurer.ConfigureWeb(context.Background(), goweb.NewRegistry()); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
}
