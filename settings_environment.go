package gbcweb

import (
	"strconv"
	"strings"

	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
)

func (s *settings) applyEnvironment(environment coreenv.Environment) error {
	if environment == nil {
		return nil
	}
	if value, ok := environment.GetProperty(PropertyApplicationName); ok {
		if err := WithApplicationName(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyServletContextPath); ok {
		if err := WithContextPath(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyServletMapping); ok {
		if err := WithMappingPattern(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesEnabled); ok {
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return arkerrors.Wrapf(arkerrors.CodeInvalidArgument, err, "invalid static resources enabled value %q", value)
		}
		if err := WithStaticResourcesEnabled(enabled)(s); err != nil {
			return err
		}
	}
	if value, ok := staticResourceLocationsProperty(environment); ok {
		if err := WithStaticResourceLocations(value)(s); err != nil {
			return err
		}
	}
	if value, ok := staticResourcePatternProperty(environment); ok {
		if err := WithStaticResourcePattern(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesServletName); ok {
		if err := WithStaticResourceServletName(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesWelcomeFiles); ok {
		if err := WithStaticResourceWelcomeFiles(value)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourcesCacheControl); ok {
		if err := WithStaticResourceCacheControl(value)(s); err != nil {
			return err
		}
	} else if value, ok := environment.GetProperty(PropertyStaticResourcesCacheMaxAge); ok {
		maxAge, err := parseDurationProperty(PropertyStaticResourcesCacheMaxAge, value)
		if err != nil {
			return err
		}
		if err := WithStaticResourceCacheMaxAge(maxAge)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourceContentVersioningEnabled); ok {
		enabled, err := parseBoolProperty(PropertyStaticResourceContentVersioningEnabled, value)
		if err != nil {
			return err
		}
		if err := WithStaticResourceContentVersioning(enabled)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertyStaticResourceFixedVersion); ok {
		if err := WithStaticResourceFixedVersion(value)(s); err != nil {
			return err
		}
	}
	if err := s.applyFilterEnvironment(environment); err != nil {
		return err
	}
	if err := s.applyViewEnvironment(environment); err != nil {
		return err
	}
	if err := s.applyHTTPClientEnvironment(environment); err != nil {
		return err
	}
	return s.applyErrorEnvironment(environment)
}
