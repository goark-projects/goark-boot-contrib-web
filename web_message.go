package gbcweb

import (
	"context"

	arkweb "goark.dev/arkarta/web"
	goarkcontainer "goark.dev/goark/container"
	goweb "goark.dev/goark/web"
	"goark.dev/goark/web/message"
)

const orderMessageIOConfigurer = -1000

// RegisterMessageConverter 注册同时参与请求读取和响应写出的消息转换器 Bean。
func RegisterMessageConverter(registry *goarkcontainer.Registry, name string, converter message.HTTPConverter, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[message.HTTPConverter](registry, name, converter, options...)
}

// RegisterMessageReadConverter 注册请求体读取转换器 Bean。
func RegisterMessageReadConverter(registry *goarkcontainer.Registry, name string, converter message.ReadConverter, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[message.ReadConverter](registry, name, converter, options...)
}

// RegisterMessageWriteConverter 注册响应体写出转换器 Bean。
func RegisterMessageWriteConverter(registry *goarkcontainer.Registry, name string, converter message.Converter, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[message.Converter](registry, name, converter, options...)
}

// RegisterRequestBodyAdvice 注册请求体读取增强器 Bean。
func RegisterRequestBodyAdvice(registry *goarkcontainer.Registry, name string, advice message.ReadAdvice, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[message.ReadAdvice](registry, name, advice, options...)
}

// RegisterResponseAdvice 注册响应体写出增强器 Bean。
func RegisterResponseAdvice(registry *goarkcontainer.Registry, name string, advice arkweb.ResponseAdvice, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[arkweb.ResponseAdvice](registry, name, advice, options...)
}

func registerMessageIO(registry *goarkcontainer.Registry) error {
	if _, exists := registry.Definition(BeanNameMessageWriter); !exists {
		if err := goarkcontainer.Register[message.Writer](registry, BeanNameMessageWriter, newMessageWriter); err != nil {
			return err
		}
	}
	if _, exists := registry.Definition(BeanNameMessageReader); !exists {
		if err := goarkcontainer.Register[message.Reader](registry, BeanNameMessageReader, newMessageReader); err != nil {
			return err
		}
	}
	if _, exists := registry.Definition(BeanNameMessageIOConfigurer); exists {
		return nil
	}
	return goarkcontainer.Register[goweb.Configurer](
		registry,
		BeanNameMessageIOConfigurer,
		newMessageIOConfigurer,
		goarkcontainer.WithOrder(orderMessageIOConfigurer),
	)
}

func newMessageWriter(ctx context.Context, resolver goarkcontainer.Resolver) (message.Writer, error) {
	converters, err := goarkcontainer.GetAllByType[message.Converter](ctx, resolver)
	if err != nil {
		return message.Writer{}, err
	}
	return message.NewWriter(message.WithPrependedConverters(converters...)), nil
}

func newMessageReader(ctx context.Context, resolver goarkcontainer.Resolver) (message.Reader, error) {
	converters, err := goarkcontainer.GetAllByType[message.ReadConverter](ctx, resolver)
	if err != nil {
		return message.Reader{}, err
	}
	advice, err := goarkcontainer.GetAllByType[message.ReadAdvice](ctx, resolver)
	if err != nil {
		return message.Reader{}, err
	}
	return message.NewReader(
		message.WithPrependedReadConverters(converters...),
		message.WithReadAdvice(advice...),
	), nil
}

func newMessageIOConfigurer(ctx context.Context, resolver goarkcontainer.Resolver) (goweb.Configurer, error) {
	reader, err := goarkcontainer.Get[message.Reader](ctx, resolver, BeanNameMessageReader)
	if err != nil {
		return nil, err
	}
	writer, err := goarkcontainer.Get[message.Writer](ctx, resolver, BeanNameMessageWriter)
	if err != nil {
		return nil, err
	}
	advice, err := goarkcontainer.GetAllByType[arkweb.ResponseAdvice](ctx, resolver)
	if err != nil {
		return nil, err
	}
	return messageIOConfigurer{
		reader:         reader,
		writer:         writer,
		responseAdvice: advice,
	}, nil
}

type messageIOConfigurer struct {
	reader         message.Reader
	writer         message.Writer
	responseAdvice []arkweb.ResponseAdvice
}

func (c messageIOConfigurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	registry.UseMessageReader(c.reader)
	registry.UseMessageWriter(c.writer)
	for _, advice := range c.responseAdvice {
		registry.UseResponseAdvice(advice)
	}
	return nil
}
