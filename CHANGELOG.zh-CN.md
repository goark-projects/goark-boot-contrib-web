# 变更日志

[English](CHANGELOG.md) | 中文

这里记录 Goark Boot Contrib Web 的重要变更。

## [未发布]

暂无未发布变更。

## [0.0.1] - 2026-09-06

### 新增

- 在嵌入式 Arkhos 容器上自动配置 Goark MVC 和 Arkarta Servlet。
- 静态资源、模板视图、消息转换器、校验、类型转换、Problem Details、WebSocket
  Endpoint、Filter 和出站 HTTP Client。
- Controller Advice、请求与响应 Advice、Interceptor、CORS、Binding Result、
  Locale、Flash Attribute 和 Session Attribute。
- 稳定 Goark 命名空间下的 Spring 风格 Web 配置属性。
- 基于 Go 1.26 的跨平台测试、vet 和 race 门禁。

### 变更

- 将所有实际使用的 `golang.org/x` 模块对齐到最新稳定版本。

### 修复

- 托管 HTTP Client 在测试和应用关闭时释放连接。
- Problem Details 保持为兜底错误映射器。
- Request Mapping、响应媒体类型、Arkhos 日志和 Server 属性遵循文档契约。

[未发布]: https://github.com/goark-projects/goark-boot-contrib-web/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/goark-projects/goark-boot-contrib-web/releases/tag/v0.0.1
