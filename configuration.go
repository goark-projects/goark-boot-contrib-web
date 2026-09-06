package gbcweb

import (
	"context"

	servletcontainer "goark.dev/arkarta/servlet/container"
	"goark.dev/arkarta/servlet/session"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	goarkcontainer "goark.dev/goark/container"
	appcontext "goark.dev/goark/context"
	goweb "goark.dev/goark/web"
	gowebcors "goark.dev/goark/web/cors"
	gowebfilter "goark.dev/goark/web/filter"
	mvcflash "goark.dev/goark/web/mvc/flash"
	mvcsessionattrs "goark.dev/goark/web/mvc/sessionattrs"
)

// AutoConfigure 创建 Web 自动配置，并默认包含 Arkhos 嵌入式容器。
func AutoConfigure(options ...Option) boot.AutoConfiguration {
	copied := append([]Option(nil), options...)
	return boot.NewAutoConfiguration(
		StarterID,
		func(ctx context.Context, app *appcontext.ApplicationContext) error {
			resolved, err := newSettings(nil, copied)
			if err != nil {
				return err
			}
			if err := registerConfigurationIfAbsent(app, configuration{options: copied}); err != nil {
				return err
			}
			return configureArkhosIfAbsent(ctx, app, resolved.arkhosOptions)
		},
	)
}

type configuration struct {
	options []Option
}

func (c configuration) Name() string {
	return StarterID + ".configuration"
}

func (c configuration) Order() int {
	return 0
}

func (c configuration) Register(ctx context.Context, registry *goarkcontainer.Registry) error {
	return c.RegisterWithContext(ctx, appcontext.NewConfigurationContext(nil, registry))
}

func (c configuration) RegisterWithContext(
	_ context.Context,
	config appcontext.ConfigurationContext,
) error {
	resolved, err := newSettings(config.Environment(), c.options)
	if err != nil {
		return err
	}
	if err := registerStaticResources(config.Registry(), resolved.staticResources); err != nil {
		return err
	}
	if err := registerViewTemplates(config.Registry(), resolved.viewTemplates); err != nil {
		return err
	}
	if err := registerHTTPClient(config.Registry(), resolved.httpClient); err != nil {
		return err
	}
	if err := registerMessageIO(config.Registry()); err != nil {
		return err
	}
	if err := registerWebValidator(config.Registry()); err != nil {
		return err
	}
	if err := registerMVCConversion(config.Registry()); err != nil {
		return err
	}
	if err := registerWebFilters(config.Registry(), resolved.filters); err != nil {
		return err
	}
	if err := registerErrorHandling(config.Registry(), resolved.errorHandling); err != nil {
		return err
	}
	return goarkcontainer.Register[*servletcontainer.Deployment](
		config.Registry(),
		BeanNameDeployment,
		func(
			ctx context.Context,
			resolver goarkcontainer.Resolver,
		) (*servletcontainer.Deployment, error) {
			registry := goweb.NewRegistry()
			if err := goweb.ApplyConfigurers(ctx, resolver, registry); err != nil {
				return nil, err
			}
			return goweb.BuildDeployment(registry, goweb.DeploymentSpec{
				AppName:           resolved.applicationName,
				ContextPath:       resolved.contextPath,
				MappingPattern:    resolved.mappingPattern,
				RouterOptions:     resolved.routerOptions,
				WebAppOptions:     resolved.webAppOptions,
				DeploymentOptions: resolved.deploymentOptions,
			})
		},
	)
}

func registerConfigurationIfAbsent(
	app *appcontext.ApplicationContext,
	configuration appcontext.Configuration,
) error {
	if hasConfiguration(app, configuration.Name()) {
		return nil
	}
	return app.RegisterConfiguration(configuration)
}

func configureArkhosIfAbsent(
	ctx context.Context,
	app *appcontext.ApplicationContext,
	options []gbcarkhos.Option,
) error {
	if hasConfiguration(app, gbcarkhos.StarterID+".configuration") {
		return nil
	}
	return gbcarkhos.AutoConfigure(options...).Configure(ctx, app)
}

func hasConfiguration(app *appcontext.ApplicationContext, name string) bool {
	for _, descriptor := range app.Configurations() {
		if descriptor.Name == name {
			return true
		}
	}
	return false
}

func registerWebFilters(registry *goarkcontainer.Registry, settings filterSettings) error {
	mvcSessionManager := mvcFilterSessionManager(settings)
	if settings.characterEncoding.enabled {
		options := append([]gowebfilter.CharacterEncodingOption{
			gowebfilter.WithCharacterEncoding(settings.characterEncoding.encoding),
			gowebfilter.WithForceRequestEncoding(settings.characterEncoding.forceRequest),
			gowebfilter.WithForceResponseEncoding(settings.characterEncoding.forceResponse),
		}, settings.characterEncoding.options...)
		if err := goweb.RegisterFilter(
			registry, BeanNameCharacterEncodingFilter, gowebfilter.CharacterEncoding(
				options...,
			), goarkcontainer.WithOrder(orderCharacterEncodingFilter)); err != nil {
			return err
		}
	}
	if settings.forwardedHeaders.enabled {
		if err := goweb.RegisterFilter(
			registry, BeanNameForwardedHeadersFilter, gowebfilter.ForwardedHeaders(),
			goarkcontainer.WithOrder(orderForwardedHeadersFilter),
		); err != nil {
			return err
		}
	}
	if settings.cors.enabled {
		filter, err := gowebcors.New(settings.cors.config)
		if err != nil {
			return err
		}
		if err := goweb.RegisterFilter(
			registry, BeanNameCORSFilter, filter,
			goarkcontainer.WithOrder(orderCORSFilter),
		); err != nil {
			return err
		}
	}
	if settings.hiddenMethod.enabled {
		if err := goweb.RegisterFilter(
			registry, BeanNameHiddenHTTPMethodFilter, gowebfilter.HiddenHTTPMethod(
				settings.hiddenMethod.options...,
			), goarkcontainer.WithOrder(orderHiddenHTTPMethodFilter)); err != nil {
			return err
		}
	}
	if settings.formContent.enabled {
		options := append([]gowebfilter.FormContentOption{
			gowebfilter.WithFormContentMaxBodyBytes(settings.formContent.maxBodyBytes),
		}, settings.formContent.options...)
		if err := goweb.RegisterFilter(registry, BeanNameFormContentFilter, gowebfilter.FormContent(
			options...,
		), goarkcontainer.WithOrder(orderFormContentFilter)); err != nil {
			return err
		}
	}
	if settings.flashMap.enabled {
		filter, err := mvcflash.NewSessionFilter(
			mvcSessionManager,
			mvcflash.WithTimeout(settings.flashMap.timeout),
		)
		if err != nil {
			return err
		}
		if err := goweb.RegisterFilter(
			registry, BeanNameFlashMapFilter, filter,
			goarkcontainer.WithOrder(orderFlashMapFilter),
		); err != nil {
			return err
		}
	}
	if settings.sessionAttributes.enabled {
		filter, err := mvcsessionattrs.NewSessionFilter(mvcSessionManager)
		if err != nil {
			return err
		}
		if err := goweb.RegisterFilter(
			registry, BeanNameSessionAttributesFilter, filter,
			goarkcontainer.WithOrder(orderSessionAttributesFilter),
		); err != nil {
			return err
		}
	}
	if settings.shallowETag.enabled {
		if err := goweb.RegisterFilter(registry, BeanNameShallowETagFilter, gowebfilter.ShallowETag(
			gowebfilter.WithMaxBodyBytes(settings.shallowETag.maxBodyBytes),
		), goarkcontainer.WithOrder(orderShallowETagFilter)); err != nil {
			return err
		}
	}
	return nil
}

func mvcFilterSessionManager(settings filterSettings) session.Manager {
	if !settings.flashMap.enabled && !settings.sessionAttributes.enabled {
		return nil
	}
	return session.NewMemoryManager()
}
