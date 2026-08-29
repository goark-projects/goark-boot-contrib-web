package gbcweb

import (
	"context"

	goarkcontainer "goark.dev/goark/container"
	"goark.dev/goark/core/convert"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/mvc"
)

const orderMVCConversionConfigurer = -900

// RegisterConverter 注册 MVC 参数转换器 Bean。
func RegisterConverter(registry *goarkcontainer.Registry, name string, converter convert.Converter, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[convert.Converter](registry, name, converter, options...)
}

func registerMVCConversion(registry *goarkcontainer.Registry) error {
	if _, exists := registry.Definition(BeanNameConversionService); !exists {
		if err := goarkcontainer.Register[*convert.Service](registry, BeanNameConversionService, newMVCConversionService); err != nil {
			return err
		}
	}
	if _, exists := registry.Definition(BeanNameConversionConfigurer); exists {
		return nil
	}
	return goarkcontainer.Register[goweb.Configurer](
		registry,
		BeanNameConversionConfigurer,
		newMVCConversionConfigurer,
		goarkcontainer.WithOrder(orderMVCConversionConfigurer),
	)
}

func newMVCConversionService(ctx context.Context, resolver goarkcontainer.Resolver) (*convert.Service, error) {
	converters, err := goarkcontainer.GetAllByType[convert.Converter](ctx, resolver)
	if err != nil {
		return nil, err
	}
	return convert.NewService(converters...)
}

func newMVCConversionConfigurer(ctx context.Context, resolver goarkcontainer.Resolver) (goweb.Configurer, error) {
	service, err := goarkcontainer.Get[*convert.Service](ctx, resolver, BeanNameConversionService)
	if err != nil {
		return nil, err
	}
	return mvcConversionConfigurer{service: service}, nil
}

type mvcConversionConfigurer struct {
	service *convert.Service
}

func (c mvcConversionConfigurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	registry.Use(mvc.ConversionInterceptor(c.service))
	return nil
}
