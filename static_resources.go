package gbcweb

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	goarkcontainer "goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
	"goark.dev/goark/web/static"
)

const (
	propertySpringStaticLocations    = "spring.web.resources.static-locations"
	propertySpringStaticPattern      = "spring.mvc.static-path-pattern"
	propertySpringStaticCacheControl = "spring.web.resources.cache.cache-control"
	propertySpringStaticCacheMaxAge  = "spring.web.resources.cache.cachecontrol.max-age"
	propertySpringStaticCachePeriod  = "spring.web.resources.cache.period"
	propertySpringStaticContentChain = "spring.web.resources.chain.strategy.content.enabled"
)

type staticResourceSettings struct {
	enabled         bool
	locations       []string
	pattern         string
	servletName     string
	welcomeFiles    []string
	welcomeFilesSet bool
	cacheControl    string
	contentVersion  bool
}

func defaultStaticResourceSettings() staticResourceSettings {
	return staticResourceSettings{
		enabled:        DefaultStaticResourcesEnabled,
		locations:      splitStaticResourceList([]string{DefaultStaticResourcesLocations}),
		pattern:        DefaultStaticResourcesPattern,
		servletName:    DefaultStaticResourcesServletName,
		contentVersion: DefaultStaticResourceContentVersioningEnabled,
	}
}

func registerStaticResources(registry *goarkcontainer.Registry, settings staticResourceSettings) error {
	if !settings.enabled {
		return nil
	}
	roots, err := openStaticResourceRoots(settings.locations)
	if err != nil {
		return err
	}
	if len(roots) == 0 {
		return nil
	}
	options := []static.Option{static.WithServletName(settings.servletName)}
	if settings.welcomeFilesSet {
		options = append(options, static.WithWelcomeFiles(settings.welcomeFiles...))
	}
	if settings.cacheControl != "" {
		options = append(options, static.WithCacheControl(settings.cacheControl))
	}
	if settings.contentVersion {
		options = append(options, static.WithContentVersioning())
	}
	return static.Register(registry, BeanNameStaticResources, settings.pattern, fallbackFS{roots: roots}, options...)
}

func openStaticResourceRoots(locations []string) ([]fs.FS, error) {
	roots := make([]fs.FS, 0, len(locations))
	for _, location := range locations {
		path, err := filepath.Abs(strings.TrimPrefix(strings.TrimSpace(location), "file:"))
		if err != nil {
			return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "failed to resolve static resource location %q", location)
		}
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "failed to inspect static resource location %q", location)
		}
		if !info.IsDir() {
			return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "static resource location %q is not a directory", location)
		}
		roots = append(roots, os.DirFS(path))
	}
	return roots, nil
}

type fallbackFS struct {
	roots []fs.FS
}

func (f fallbackFS) Open(name string) (fs.File, error) {
	var last error
	for _, root := range f.roots {
		file, err := root.Open(name)
		if err == nil {
			return file, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			last = err
			continue
		}
		return nil, err
	}
	if last != nil {
		return nil, last
	}
	return nil, fs.ErrNotExist
}

func normalizeStaticResourceLocations(locations []string) ([]string, error) {
	items := splitStaticResourceList(locations)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, normalizeStaticResourceLocation(item))
	}
	if len(out) == 0 {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "static resource locations are empty")
	}
	return out, nil
}

func normalizeStaticResourceLocation(location string) string {
	location = strings.TrimSpace(location)
	switch {
	case strings.HasPrefix(location, "classpath:"):
		path := strings.TrimPrefix(location, "classpath:")
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			return "resource"
		}
		return filepath.Join("resource", filepath.FromSlash(path))
	case strings.HasPrefix(location, "file:"):
		return strings.TrimPrefix(location, "file:")
	default:
		return location
	}
}

func splitStaticResourceList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}

func staticResourceLocationsProperty(environment coreenv.Environment) (string, bool) {
	if value, ok := environment.GetProperty(PropertyStaticResourcesLocations); ok {
		return value, true
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesLocation); ok {
		return value, true
	}
	return environment.GetProperty(propertySpringStaticLocations)
}

func staticResourcePatternProperty(environment coreenv.Environment) (string, bool) {
	if value, ok := environment.GetProperty(PropertyStaticResourcesPattern); ok {
		return value, true
	}
	return environment.GetProperty(propertySpringStaticPattern)
}
