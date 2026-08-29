package gbcweb

import (
	"context"

	goarkcontainer "goark.dev/goark/container"
	arkerrors "goark.dev/goark/errors"
	webclient "goark.dev/goark/web/client"
)

// HTTPClientBuilderCustomizer 定制默认 Web HTTP 客户端构建器。
type HTTPClientBuilderCustomizer interface {
	CustomizeHTTPClientBuilder(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error)
}

// HTTPClientBuilderCustomizerFunc 将函数适配为 HTTP 客户端构建器定制器。
type HTTPClientBuilderCustomizerFunc func(context.Context, *webclient.Builder) (*webclient.Builder, error)

// CustomizeHTTPClientBuilder 执行函数型定制器。
func (f HTTPClientBuilderCustomizerFunc) CustomizeHTTPClientBuilder(ctx context.Context, builder *webclient.Builder) (*webclient.Builder, error) {
	if f == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "web http client builder customizer is nil")
	}
	return f(ctx, builder)
}

// RegisterHTTPClientBuilderCustomizer 注册 Web HTTP 客户端构建器定制器 Bean。
func RegisterHTTPClientBuilderCustomizer(registry *goarkcontainer.Registry, name string, customizer HTTPClientBuilderCustomizer, options ...goarkcontainer.Option) error {
	return goarkcontainer.RegisterInstance[HTTPClientBuilderCustomizer](registry, name, customizer, options...)
}

func newHTTPClientBuilder(ctx context.Context, resolver goarkcontainer.Resolver, options []webclient.Option) (*webclient.Builder, error) {
	builder := webclient.NewBuilder(options...)
	return applyHTTPClientBuilderCustomizers(ctx, resolver, builder)
}

func applyHTTPClientBuilderCustomizers(ctx context.Context, resolver goarkcontainer.Resolver, builder *webclient.Builder) (*webclient.Builder, error) {
	if builder == nil {
		return nil, arkerrors.New(arkerrors.CodeCreation, "web http client builder is nil")
	}
	customizers, err := goarkcontainer.GetAllByType[HTTPClientBuilderCustomizer](ctx, resolver)
	if err != nil {
		return nil, err
	}
	for _, customizer := range customizers {
		if customizer == nil {
			continue
		}
		next, err := customizer.CustomizeHTTPClientBuilder(ctx, builder)
		if err != nil {
			return nil, err
		}
		if next == nil {
			return nil, arkerrors.New(arkerrors.CodeCreation, "web http client builder customizer returned nil")
		}
		builder = next
	}
	return builder, nil
}
