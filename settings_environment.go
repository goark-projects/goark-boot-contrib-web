package gbcweb

import (
	"strconv"
	"strings"
	"time"

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
			return arkerrors.Wrapf(
				arkerrors.CodeInvalidArgument,
				err,
				"invalid static resources enabled value %q",
				value,
			)
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

func (s *settings) applyFilterEnvironment(environment coreenv.Environment) error {
	if value, ok := environment.GetProperty(PropertyCORSEnabled); ok {
		enabled, err := parseBoolProperty(PropertyCORSEnabled, value)
		if err != nil {
			return err
		}
		s.filters.cors.enabled = enabled
		s.filters.cors.enabledSet = true
	}
	if value, ok := environment.GetProperty(PropertyCORSAllowedOrigins); ok {
		s.filters.cors.config.AllowedOrigins = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyCORSAllowedOriginPatterns); ok {
		s.filters.cors.config.AllowedOriginPatterns = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyCORSAllowedMethods); ok {
		s.filters.cors.config.AllowedMethods = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyCORSAllowedHeaders); ok {
		s.filters.cors.config.AllowedHeaders = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyCORSExposedHeaders); ok {
		s.filters.cors.config.ExposedHeaders = splitStaticResourceList([]string{value})
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyCORSAllowCredentials); ok {
		enabled, err := parseBoolProperty(PropertyCORSAllowCredentials, value)
		if err != nil {
			return err
		}
		s.filters.cors.config.AllowCredentials = enabled
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyCORSMaxAge); ok {
		maxAge, err := parseDurationProperty(PropertyCORSMaxAge, value)
		if err != nil {
			return err
		}
		s.filters.cors.config.MaxAge = maxAge
		enableCORSByConfiguration(&s.filters.cors)
	}
	if value, ok := environment.GetProperty(PropertyForwardedHeadersEnabled); ok {
		enabled, err := parseBoolProperty(PropertyForwardedHeadersEnabled, value)
		if err != nil {
			return err
		}
		s.filters.forwardedHeaders.enabled = enabled
	} else if value, ok := environment.GetProperty(propertyServerForwardHeadersStrategy); ok {
		s.filters.forwardedHeaders.enabled = strings.EqualFold(strings.TrimSpace(value), "framework")
	}
	if value, ok := environment.GetProperty(PropertyCharacterEncodingFilterEnabled); ok {
		enabled, err := parseBoolProperty(PropertyCharacterEncodingFilterEnabled, value)
		if err != nil {
			return err
		}
		s.filters.characterEncoding.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyCharacterEncoding); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			s.filters.characterEncoding.encoding = value
		}
	}
	if value, ok := environment.GetProperty(PropertyForceCharacterEncoding); ok {
		force, err := parseBoolProperty(PropertyForceCharacterEncoding, value)
		if err != nil {
			return err
		}
		s.filters.characterEncoding.forceRequest = force
		s.filters.characterEncoding.forceResponse = force
	}
	if value, ok := environment.GetProperty(PropertyForceRequestCharacterEncoding); ok {
		force, err := parseBoolProperty(PropertyForceRequestCharacterEncoding, value)
		if err != nil {
			return err
		}
		s.filters.characterEncoding.forceRequest = force
	}
	if value, ok := environment.GetProperty(PropertyForceResponseCharacterEncoding); ok {
		force, err := parseBoolProperty(PropertyForceResponseCharacterEncoding, value)
		if err != nil {
			return err
		}
		s.filters.characterEncoding.forceResponse = force
	}
	if value, ok := environment.GetProperty(PropertyShallowETagEnabled); ok {
		enabled, err := parseBoolProperty(PropertyShallowETagEnabled, value)
		if err != nil {
			return err
		}
		s.filters.shallowETag.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyShallowETagMaxBodyBytes); ok {
		size, err := parseInt64Property(PropertyShallowETagMaxBodyBytes, value)
		if err != nil {
			return err
		}
		s.filters.shallowETag.maxBodyBytes = size
	}
	if value, ok := environment.GetProperty(PropertyHiddenHTTPMethodFilterEnabled); ok {
		enabled, err := parseBoolProperty(PropertyHiddenHTTPMethodFilterEnabled, value)
		if err != nil {
			return err
		}
		s.filters.hiddenMethod.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyFormContentFilterEnabled); ok {
		enabled, err := parseBoolProperty(PropertyFormContentFilterEnabled, value)
		if err != nil {
			return err
		}
		s.filters.formContent.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyFormContentMaxBodyBytes); ok {
		size, err := parseInt64Property(PropertyFormContentMaxBodyBytes, value)
		if err != nil {
			return err
		}
		s.filters.formContent.maxBodyBytes = size
	}
	if value, ok := environment.GetProperty(PropertyFlashMapFilterEnabled); ok {
		enabled, err := parseBoolProperty(PropertyFlashMapFilterEnabled, value)
		if err != nil {
			return err
		}
		s.filters.flashMap.enabled = enabled
	}
	if value, ok := environment.GetProperty(PropertyFlashMapTimeout); ok &&
		strings.TrimSpace(value) != "" {
		timeout, err := parseDurationProperty(PropertyFlashMapTimeout, value)
		if err != nil {
			return err
		}
		if err := WithFlashMapTimeout(timeout)(s); err != nil {
			return err
		}
	}
	if value, ok := environment.GetProperty(PropertySessionAttributesFilterEnabled); ok {
		enabled, err := parseBoolProperty(PropertySessionAttributesFilterEnabled, value)
		if err != nil {
			return err
		}
		s.filters.sessionAttributes.enabled = enabled
	}
	return nil
}

func enableCORSByConfiguration(settings *corsFilterSettings) {
	if !settings.enabledSet {
		settings.enabled = true
	}
}

func firstProperty(environment coreenv.Environment, names ...string) (string, bool) {
	for _, name := range names {
		if value, ok := environment.GetProperty(name); ok {
			return value, true
		}
	}
	return "", false
}

func parseBoolProperty(name string, value string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, arkerrors.Wrapf(
			arkerrors.CodeInvalidArgument,
			err,
			"invalid boolean property %s=%q",
			name,
			value,
		)
	}
	return parsed, nil
}

func parseDurationProperty(name string, value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, arkerrors.Wrapf(
			arkerrors.CodeInvalidArgument,
			err,
			"invalid duration property %s=%q",
			name,
			value,
		)
	}
	return parsed, nil
}

func parseInt64Property(name string, value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, arkerrors.Wrapf(
			arkerrors.CodeInvalidArgument,
			err,
			"invalid int64 property %s=%q",
			name,
			value,
		)
	}
	if parsed < 0 {
		return 0, arkerrors.Newf(
			arkerrors.CodeInvalidArgument,
			"property %s=%q must be >= 0",
			name,
			value,
		)
	}
	return parsed, nil
}
