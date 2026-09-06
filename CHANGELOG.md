# Changelog

English | [中文](CHANGELOG.zh-CN.md)

All notable changes to Goark Boot Contrib Web are recorded here.

## [Unreleased]

No unreleased changes.

## [0.0.1] - 2026-09-06

### Added

- Goark MVC and Arkarta Servlet auto-configuration on the embedded Arkhos
  container.
- Static resources, template views, message converters, validation, conversion,
  Problem Details, WebSocket endpoints, filters, and outbound HTTP clients.
- Controller advice, request and response advice, interceptors, CORS, binding
  results, locale support, flash attributes, and session attributes.
- Spring-style Web configuration properties under stable Goark namespaces.
- Cross-platform CI with Go 1.26 tests, vet, and race gates.

### Fixed

- Managed HTTP clients close connections in tests and application shutdown.
- Problem Details remains the fallback error mapper.
- Request mapping, response media type, Arkhos logging, and server-property
  behavior follow the documented contracts.

[Unreleased]: https://github.com/goark-projects/goark-boot-contrib-web/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/goark-projects/goark-boot-contrib-web/releases/tag/v0.0.1
