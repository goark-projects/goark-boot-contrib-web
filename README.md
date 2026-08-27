# Goark Boot Contrib Web

Official Goark Boot starter module for web applications.

## Module

- Module path: `goark.dev/gbc-web`
- Repository: `github.com/goark-projects/goark-boot-contrib-web`
- License: Apache-2.0
- Default branch: `main`
- Development branch: `dev`

## Scope

`goark.dev/gbc-web` is the Goark-managed web starter. It is intended to provide the Spring Boot `spring-boot-starter-web` equivalent for the Goark ecosystem:

- Goark Web and MVC defaults.
- Arkarta web and servlet integration.
- Arkhos as the default embedded web container through `goark.dev/gbc-arkhos`.
- JSON binding, validation, error mapping, and handler registration conventions.

The initial repository bootstrap exposes stable module metadata only. Runtime auto-configuration APIs will be added in later implementation slices.

## Development

```bash
go test ./...
```

## Chinese

# Goark Boot Contrib Web（中文）

Goark 官方维护的 Web 应用启动器模块。

## 模块信息

- 模块路径：`goark.dev/gbc-web`
- 仓库地址：`github.com/goark-projects/goark-boot-contrib-web`
- 开源协议：Apache-2.0
- 默认分支：`main`
- 开发分支：`dev`

## 职责边界

`goark.dev/gbc-web` 是 Goark 生态中对标 Spring Boot `spring-boot-starter-web` 的官方 Web 启动器：

- 提供 Goark Web 与 MVC 默认配置。
- 集成 Arkarta Web 与 Servlet 标准。
- 默认通过 `goark.dev/gbc-arkhos` 接入 Arkhos 嵌入式 Web 容器。
- 提供 JSON 绑定、校验、错误映射和处理器注册约定。

当前初始化版本只暴露稳定的模块元数据。运行期自动配置 API 会在后续实现切片中补齐。

## 开发

```bash
go test ./...
```
