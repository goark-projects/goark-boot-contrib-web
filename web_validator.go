package gbcweb

import (
	"context"

	"goark.dev/arkarta/validation"
	goarkcontainer "goark.dev/goark/container"
	"goark.dev/goark/core/util"
	goweb "goark.dev/goark/web"
)

const (
	orderWebValidatorConfigurer = -950
	defaultValidatorPriority    = 10000
	customValidatorPriority     = 0
)

// RegisterValidator 注册 Web 请求校验器 Bean。
func RegisterValidator(registry *goarkcontainer.Registry, name string, validator validation.Validator, options ...goarkcontainer.Option) error {
	if util.IsNil(validator) {
		return goweb.ErrNilValidator
	}
	validatorOptions := make([]goarkcontainer.Option, 0, len(options)+1)
	validatorOptions = append(validatorOptions, goarkcontainer.WithPriority(customValidatorPriority))
	validatorOptions = append(validatorOptions, options...)
	return goarkcontainer.RegisterInstance[validation.Validator](registry, name, validator, validatorOptions...)
}

func registerWebValidator(registry *goarkcontainer.Registry) error {
	if _, exists := registry.Definition(BeanNameValidator); !exists {
		if err := goarkcontainer.RegisterInstance[validation.Validator](
			registry,
			BeanNameValidator,
			validation.NewValidator(),
			goarkcontainer.WithPriority(defaultValidatorPriority),
		); err != nil {
			return err
		}
	}
	if _, exists := registry.Definition(BeanNameValidatorConfigurer); exists {
		return nil
	}
	return goarkcontainer.Register[goweb.Configurer](
		registry,
		BeanNameValidatorConfigurer,
		newWebValidatorConfigurer,
		goarkcontainer.WithOrder(orderWebValidatorConfigurer),
	)
}

func newWebValidatorConfigurer(ctx context.Context, resolver goarkcontainer.Resolver) (goweb.Configurer, error) {
	validator, err := goarkcontainer.GetByType[validation.Validator](ctx, resolver)
	if err != nil {
		return nil, err
	}
	return webValidatorConfigurer{validator: validator}, nil
}

type webValidatorConfigurer struct {
	validator validation.Validator
}

func (c webValidatorConfigurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	registry.UseValidator(c.validator)
	return nil
}
