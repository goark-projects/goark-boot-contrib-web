package gbcweb

import (
	"context"

	servletcontainer "goark.dev/arkarta/servlet/container"
	"goark.dev/boot"
	gbcarkhos "goark.dev/gbc-arkhos"
	goarkcontainer "goark.dev/goark/container"
	appcontext "goark.dev/goark/context"
	goweb "goark.dev/goark/web"
)

// AutoConfigure 创建 Web 自动配置，并默认包含 Arkhos 嵌入式容器。
func AutoConfigure(options ...Option) boot.AutoConfiguration {
	copied := append([]Option(nil), options...)
	return boot.NewAutoConfiguration(StarterID, func(ctx context.Context, app *appcontext.ApplicationContext) error {
		resolved, err := newSettings(nil, copied)
		if err != nil {
			return err
		}
		if err := registerConfigurationIfAbsent(app, configuration{options: copied}); err != nil {
			return err
		}
		return configureArkhosIfAbsent(ctx, app, resolved.arkhosOptions)
	})
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

func (c configuration) RegisterWithContext(_ context.Context, config appcontext.ConfigurationContext) error {
	resolved, err := newSettings(config.Environment(), c.options)
	if err != nil {
		return err
	}
	if err := registerStaticResources(config.Registry(), resolved.staticResources); err != nil {
		return err
	}
	return goarkcontainer.Register[*servletcontainer.Deployment](
		config.Registry(),
		BeanNameDeployment,
		func(ctx context.Context, resolver goarkcontainer.Resolver) (*servletcontainer.Deployment, error) {
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

func registerConfigurationIfAbsent(app *appcontext.ApplicationContext, configuration appcontext.Configuration) error {
	if hasConfiguration(app, configuration.Name()) {
		return nil
	}
	return app.RegisterConfiguration(configuration)
}

func configureArkhosIfAbsent(ctx context.Context, app *appcontext.ApplicationContext, options []gbcarkhos.Option) error {
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
