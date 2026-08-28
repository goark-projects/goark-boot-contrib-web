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
- JSON binding and handler registration conventions supplied by `goark.dev/goark/web/mvc`.

## Usage

```go
package main

import (
	"context"

	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
)

func main() {
	app, err := boot.Run(context.Background(),
		boot.WithConfigDataOptions(configdata.WithLocations("config")),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		panic(err)
	}
	defer app.Close(context.Background())
}
```

Business packages contribute MVC controllers or `goark.dev/goark/web.Configurer` beans.
This starter builds the default Arkarta Servlet deployment and starts Arkhos through the Boot lifecycle.

## Configuration Properties

| Property | Default | Description |
| --- | --- | --- |
| `goark.web.application.name` | `goark` | Arkarta Servlet web application name. |
| `goark.web.servlet.context-path` | `/` | Servlet context path. Must start with `/`. |
| `goark.web.servlet.mapping` | `/` | Servlet mapping pattern for the Goark Web router. |

Server and multipart properties are provided by `goark.dev/gbc-arkhos`, which this starter includes by default.

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
- 提供 `goark.dev/goark/web/mvc` 的 JSON 绑定和处理器注册约定。

## 使用方式

业务包只需要贡献 MVC 控制器或 `goark.dev/goark/web.Configurer` Bean。
本启动器会构建默认 Arkarta Servlet 部署，并通过 Boot 生命周期启动 Arkhos。

## 配置属性

| 属性 | 默认值 | 说明 |
| --- | --- | --- |
| `goark.web.application.name` | `goark` | Arkarta Servlet Web 应用名称。 |
| `goark.web.servlet.context-path` | `/` | Servlet 上下文路径，必须以 `/` 开头。 |
| `goark.web.servlet.mapping` | `/` | Goark Web Router 对应的 Servlet 映射模式。 |

服务端监听和 multipart 属性由默认包含的 `goark.dev/gbc-arkhos` 提供。

## 开发

```bash
go test ./...
```
