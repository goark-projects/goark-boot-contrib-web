package gbcweb

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	goarkcontainer "goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	goweb "goark.dev/goark/web"
	mvcview "goark.dev/goark/web/mvc/view"
)

const orderViewInterceptor = -50

type viewTemplateSettings struct {
	enabled     bool
	location    string
	prefix      string
	suffix      string
	contentType string
}

func defaultViewTemplateSettings() viewTemplateSettings {
	return viewTemplateSettings{
		enabled:     DefaultViewTemplatesEnabled,
		location:    DefaultViewTemplatesLocation,
		suffix:      DefaultViewTemplateSuffix,
		contentType: DefaultViewTemplateContentType,
	}
}

func (s *settings) applyViewEnvironment(environment coreenv.Environment) error {
	if value, ok := environment.GetProperty(PropertyViewTemplatesEnabled); ok {
		enabled, err := parseBoolProperty(PropertyViewTemplatesEnabled, value)
		if err != nil {
			return err
		}
		s.viewTemplates.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyViewTemplatesLocation); ok {
		if err := WithViewTemplatesLocation(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyViewTemplatePrefix); ok {
		if err := WithViewTemplatePrefix(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyViewTemplateSuffix); ok {
		if err := WithViewTemplateSuffix(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyViewTemplateContentType); ok {
		if err := WithViewTemplateContentType(value)(s); err != nil {
			return err
		}
	}
	return nil
}

func registerViewTemplates(registry *goarkcontainer.Registry, settings viewTemplateSettings) error {
	if !settings.enabled {
		return nil
	}
	root, err := openViewTemplateRoot(settings.location)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	resolver, err := mvcview.NewTemplateResolver(root,
		mvcview.WithPrefix(settings.prefix),
		mvcview.WithSuffix(settings.suffix),
		mvcview.WithTemplateContentType(settings.contentType),
	)
	if errors.Is(err, mvcview.ErrNoTemplates) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := goarkcontainer.RegisterInstance[mvcview.Resolver](registry, BeanNameViewResolver, resolver); err != nil {
		return err
	}
	return goweb.RegisterConfigurer(registry, BeanNameViewInterceptor, viewTemplateConfigurer{resolver: resolver}, goarkcontainer.WithOrder(orderViewInterceptor))
}

func openViewTemplateRoot(location string) (fs.FS, error) {
	path, err := filepath.Abs(strings.TrimSpace(location))
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "failed to resolve view template location %q", location)
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fs.ErrNotExist
	}
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "failed to inspect view template location %q", location)
	}
	if !info.IsDir() {
		return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "view template location %q is not a directory", location)
	}
	return os.DirFS(path), nil
}

type viewTemplateConfigurer struct {
	resolver mvcview.Resolver
}

func (c viewTemplateConfigurer) ConfigureWeb(ctx context.Context, registry *goweb.Registry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if registry == nil {
		return goweb.ErrNilRegistry
	}
	registry.Use(mvcview.Interceptor(c.resolver))
	return nil
}
