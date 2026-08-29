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
- Auto-application of `goark.dev/goark/web.Configurer` beans, including MVC routes, response advice, and error mappers supplied by application code.
- Goark Web result conventions such as `ResponseEntity[T]`, file download, streaming responses, and Server-Sent Events.
- Conditional request support for ETag and `Last-Modified` based 304 responses.
- Path-scoped Web interceptors and Servlet filters for Spring MVC style include/exclude request chains.
- Goark MVC request mapping conditions, including consumes/produces negotiation for JSON-compatible vendor media types.
- Default outbound `goark.dev/goark/web/client` Builder and Client beans for Spring-style JSON, form, and multipart web client usage.
- Ordered `HTTPClientBuilderCustomizer` extension points for application and starter-level outbound client defaults.

## Usage

```go
package main

import (
	"context"

	"goark.dev/boot"
	"goark.dev/gbc-web"
)

func main() {
	app, err := boot.Run(context.Background(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
	)
	if err != nil {
		panic(err)
	}
	defer app.Close(context.Background())
}
```

Business packages contribute MVC controllers, response advice, error mappers, or other `goark.dev/goark/web.Configurer` beans.
This starter builds the default Arkarta Servlet deployment and starts Arkhos through the Boot lifecycle.
MVC routes may return `goark.dev/goark/web.ResponseEntity[T]`, `Download`, `TextStream`, `BinaryStream`, `SSE`, or `Redirect`; the response status, headers, cookies, cache headers, conditional validators, JSON body, negotiated `produces` media type, redirect location, and streaming headers are preserved by the generated route and starter deployment.
Routes can call `goark.dev/goark/web.CheckNotModified` before building an expensive body to honor `If-None-Match` and `If-Modified-Since` requests.
Applications may also register `gbcweb.HTTPClientBuilderCustomizer` beans to add default outbound headers, cookies, interceptors, codecs, or transports to the shared `goark.dev/goark/web/client.Builder`.

## Configuration Properties

| Property | Default | Description |
| --- | --- | --- |
| `goark.web.application.name` | `goark` | Arkarta Servlet web application name. |
| `goark.web.servlet.context-path` | `/` | Servlet context path. Must start with `/`. |
| `goark.web.servlet.mapping` | `/` | Servlet mapping pattern for the Goark Web router. |
| `goark.web.resources.static.enabled` | `true` | Enable the default static resource convention. |
| `goark.web.resources.static.locations` | `resource/static,resource/public,resource/resources,resource/META-INF/resources` | Comma-separated static resource directories. `classpath:/static/` maps to `resource/static`. |
| `goark.web.resources.static.location` | unset | Single-location alias for `goark.web.resources.static.locations`. |
| `goark.web.resources.static.pattern` | `/static/*` | Servlet mapping pattern for static resources. |
| `goark.web.resources.static.servlet-name` | `goark.boot.web.static` | Default servlet name for static resources. |
| `goark.web.resources.static.welcome-files` | `index.html,index.htm` | Comma-separated welcome files. Empty option disables welcome lookup. |
| `goark.web.resources.static.cache-control` | unset | Static resource `Cache-Control` header value. |
| `goark.web.resources.static.cache.max-age` | unset | Static resource cache duration; emits `public, max-age=N`. |
| `goark.web.resources.static.chain.content.enabled` | `false` | Enable content-hash versioned static resource paths. |
| `goark.web.mvc.view.enabled` | `true` | Enable default MVC template view resolver when templates exist. |
| `goark.web.mvc.view.templates.location` | `resource/templates` | Template root directory. `classpath:/templates/` maps to `resource/templates`. |
| `goark.web.mvc.view.prefix` | unset | Prefix added to logical view names. |
| `goark.web.mvc.view.suffix` | `.html` | Suffix added to logical view names. |
| `goark.web.mvc.view.content-type` | `text/html; charset=utf-8` | Default template response content type. |
| `goark.web.client.enabled` | `true` | Register the default outbound Web HTTP client builder and client beans. |
| `goark.web.client.base-url` | unset | Optional base URL used by relative outbound HTTP requests. |
| `goark.web.client.timeout` | `30s` | Default outbound HTTP request timeout. |
| `goark.web.client.max-response-bytes` | `16777216` | Maximum response body snapshot size. `-1` disables the limit. |
| `goark.web.client.default-headers` | unset | Comma-separated default headers in `Name=Value` format. |

Spring-compatible aliases are recognized for static resources:

| Spring-compatible property | Goark property |
| --- | --- |
| `spring.web.resources.static-locations` | `goark.web.resources.static.locations` |
| `spring.mvc.static-path-pattern` | `goark.web.resources.static.pattern` |
| `spring.web.resources.cache.cache-control` | `goark.web.resources.static.cache-control` |
| `spring.web.resources.cache.cachecontrol.max-age` | `goark.web.resources.static.cache.max-age` |
| `spring.web.resources.cache.period` | `goark.web.resources.static.cache.max-age` |
| `spring.web.resources.chain.strategy.content.enabled` | `goark.web.resources.static.chain.content.enabled` |
| `spring.mvc.view.prefix` | `goark.web.mvc.view.prefix` |
| `spring.mvc.view.suffix` | `goark.web.mvc.view.suffix` |

Server, multipart, and Servlet async timeout properties are provided by `goark.dev/gbc-arkhos`, which this starter includes by default.

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
- 自动应用业务贡献的 `goark.dev/goark/web.Configurer` Bean，包括 MVC 路由、响应增强器和错误映射器。
- 支持 Goark Web 的 `ResponseEntity[T]`、文件下载、流式响应和 Server-Sent Events 结果约定。
- 支持基于 ETag 和 `Last-Modified` 的条件请求，命中时返回 304。
- 支持带路径作用域的 Web 拦截器和 Servlet 过滤器，对齐 Spring MVC 风格 include/exclude 请求链。
- 支持 Goark MVC 请求映射条件，包括 consumes/produces 以及 JSON 兼容厂商媒体类型协商。
- 提供默认出站 `goark.dev/goark/web/client` Builder 与 Client Bean，对齐 Spring 风格 JSON、表单和 multipart Web 客户端用法。
- 提供有序 `HTTPClientBuilderCustomizer` 扩展点，支持业务和其他 starter 定制出站客户端默认行为。

## 使用方式

业务包只需要贡献 MVC 控制器、响应增强器、错误映射器或其他 `goark.dev/goark/web.Configurer` Bean。
本启动器会构建默认 Arkarta Servlet 部署，并通过 Boot 生命周期启动 Arkhos。
MVC 路由可以返回 `goark.dev/goark/web.ResponseEntity[T]`、`Download`、`TextStream`、`BinaryStream`、`SSE` 或 `Redirect`；生成路由和本启动器部署链路会保留响应状态码、响应头、Cookie、缓存头、条件请求校验器、JSON 响应体、协商后的 `produces` 媒体类型、重定向 Location 和流式响应头。
路由可以在构造高成本响应体前调用 `goark.dev/goark/web.CheckNotModified`，以处理 `If-None-Match` 和 `If-Modified-Since` 请求。
业务也可以注册 `gbcweb.HTTPClientBuilderCustomizer` Bean，为共享 `goark.dev/goark/web/client.Builder` 添加默认出站请求头、Cookie、拦截器、编解码器或底层传输。

## 配置属性

| 属性 | 默认值 | 说明 |
| --- | --- | --- |
| `goark.web.application.name` | `goark` | Arkarta Servlet Web 应用名称。 |
| `goark.web.servlet.context-path` | `/` | Servlet 上下文路径，必须以 `/` 开头。 |
| `goark.web.servlet.mapping` | `/` | Goark Web Router 对应的 Servlet 映射模式。 |
| `goark.web.resources.static.enabled` | `true` | 是否启用默认静态资源约定。 |
| `goark.web.resources.static.locations` | `resource/static,resource/public,resource/resources,resource/META-INF/resources` | 静态资源目录列表，逗号分隔。`classpath:/static/` 会映射为 `resource/static`。 |
| `goark.web.resources.static.location` | 未设置 | `goark.web.resources.static.locations` 的单目录别名。 |
| `goark.web.resources.static.pattern` | `/static/*` | 静态资源 Servlet 映射。 |
| `goark.web.resources.static.servlet-name` | `goark.boot.web.static` | 静态资源 default servlet 名称。 |
| `goark.web.resources.static.welcome-files` | `index.html,index.htm` | welcome file 列表，逗号分隔；通过 Option 传空列表可禁用 welcome 查找。 |
| `goark.web.resources.static.cache-control` | 未设置 | 静态资源 `Cache-Control` 响应头。 |
| `goark.web.resources.static.cache.max-age` | 未设置 | 静态资源缓存时间，生成 `public, max-age=N`。 |
| `goark.web.resources.static.chain.content.enabled` | `false` | 是否启用内容哈希版本静态资源路径。 |
| `goark.web.mvc.view.enabled` | `true` | 模板存在时启用默认 MVC 模板视图解析器。 |
| `goark.web.mvc.view.templates.location` | `resource/templates` | 模板根目录，`classpath:/templates/` 会映射为 `resource/templates`。 |
| `goark.web.mvc.view.prefix` | 未设置 | 逻辑视图名前缀。 |
| `goark.web.mvc.view.suffix` | `.html` | 逻辑视图名后缀。 |
| `goark.web.mvc.view.content-type` | `text/html; charset=utf-8` | 模板响应默认媒体类型。 |
| `goark.web.client.enabled` | `true` | 是否注册默认出站 Web HTTP 客户端 Builder 与 Client Bean。 |
| `goark.web.client.base-url` | 未设置 | 相对出站 HTTP 请求使用的可选基础 URL。 |
| `goark.web.client.timeout` | `30s` | 默认出站 HTTP 请求超时。 |
| `goark.web.client.max-response-bytes` | `16777216` | 响应体快照最大读取字节数；`-1` 表示不限制。 |
| `goark.web.client.default-headers` | 未设置 | 默认请求头，多个值以英文逗号分隔，格式为 `Name=Value`。 |

静态资源支持以下 Spring 兼容属性：

| Spring 兼容属性 | Goark 属性 |
| --- | --- |
| `spring.web.resources.static-locations` | `goark.web.resources.static.locations` |
| `spring.mvc.static-path-pattern` | `goark.web.resources.static.pattern` |
| `spring.web.resources.cache.cache-control` | `goark.web.resources.static.cache-control` |
| `spring.web.resources.cache.cachecontrol.max-age` | `goark.web.resources.static.cache.max-age` |
| `spring.web.resources.cache.period` | `goark.web.resources.static.cache.max-age` |
| `spring.web.resources.chain.strategy.content.enabled` | `goark.web.resources.static.chain.content.enabled` |
| `spring.mvc.view.prefix` | `goark.web.mvc.view.prefix` |
| `spring.mvc.view.suffix` | `goark.web.mvc.view.suffix` |

服务端监听、multipart 和 Servlet async 超时属性由默认包含的 `goark.dev/gbc-arkhos` 提供。

## 开发

```bash
go test ./...
```
