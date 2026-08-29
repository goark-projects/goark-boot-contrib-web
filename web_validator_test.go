package gbcweb

import (
	"context"
	"testing"

	"goark.dev/arkarta/validation"
	goarkcontainer "goark.dev/goark/container"
	goweb "goark.dev/goark/web"
)

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

	validator, err := goarkcontainer.Get[validation.Validator](context.Background(), runtimeContainer, BeanNameValidator)
	if err != nil {
		t.Fatalf("resolve default validator failed: %v", err)
	}
	if validator == nil {
		t.Fatal("validator is nil")
	}
	configurer, err := goarkcontainer.Get[goweb.Configurer](context.Background(), runtimeContainer, BeanNameValidatorConfigurer)
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
	if err := RegisterValidator(registry, "testRejectingValidator", testRejectingValidator{}); err != nil {
		t.Fatalf("RegisterValidator failed: %v", err)
	}
	if err := registerWebValidator(registry); err != nil {
		t.Fatalf("registerWebValidator failed: %v", err)
	}
	runtimeContainer, err := goarkcontainer.New(registry)
	if err != nil {
		t.Fatalf("container.New failed: %v", err)
	}

	validator, err := goarkcontainer.GetByType[validation.Validator](context.Background(), runtimeContainer)
	if err != nil {
		t.Fatalf("resolve validator by type failed: %v", err)
	}
	if _, ok := validator.(testRejectingValidator); !ok {
		t.Fatalf("validator = %T, want testRejectingValidator", validator)
	}
	configurer, err := goarkcontainer.Get[goweb.Configurer](context.Background(), runtimeContainer, BeanNameValidatorConfigurer)
	if err != nil {
		t.Fatalf("resolve validator configurer failed: %v", err)
	}
	webRegistry := goweb.NewRegistry()
	if err := configurer.ConfigureWeb(context.Background(), webRegistry); err != nil {
		t.Fatalf("ConfigureWeb failed: %v", err)
	}
	if _, ok := webRegistry.Validator().(testRejectingValidator); !ok {
		t.Fatalf("web registry validator = %T, want testRejectingValidator", webRegistry.Validator())
	}
}
