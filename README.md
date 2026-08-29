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
- Spring-style request entity binding through `RequestEntity[T]`, including body, headers, method, URL, and content length metadata.
- Spring-style response Cookie construction through `ResponseCookie` and `ResponseEntity`.
- Conditional request support for ETag and `Last-Modified` based 304 responses.
- Path-scoped Web interceptors and Servlet filters for Spring MVC style include/exclude request chains.
- MVC `Controller` and `RestController` default return value semantics: views for normal controllers, response bodies for REST controllers.
- MVC `ResponseBody` route wrapper for controller methods that explicitly write normal return values as response bodies.
- MVC `Model` and `ModelAndView` return values for Spring-style server-side template controllers.
- MVC `ResponseStatus` method-level default status handling for ordinary return values and no-body handlers.
- MVC `RequestMapping` routes without HTTP method narrowing, expanded to the standard MVC request method set.
- MVC route and controller scoped CORS mappings for Spring-style `@CrossOrigin` behavior.
- MVC explicit validation groups for request body, multipart body, and model attribute binding.
- MVC typed JSON request part binding with validation groups for Spring-style multipart metadata plus file upload requests.
- MVC request parameter and request header map binding for Spring-style `Map` and multi-value map handler parameters.
- MVC `ControllerAdvice` and `RestControllerAdvice` exception handling with the same default return value semantics.
- Default UTF-8 character encoding filter with Spring Boot compatible servlet encoding properties.
- Optional hidden HTTP method filter for HTML form method override.
- Default form content filter for PUT, PATCH, and DELETE URL-encoded request parameters.
- WebSocket Endpoint registration backed by Arkarta Servlet Upgrade and the default Arkhos embedded container.
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
Handlers can bind `goark.dev/goark/web.RequestEntity[T]` when they need the converted request body together with headers, method, full URL, path, and content length metadata.
Routes built with `mvc.Return` follow the containing controller: `mvc.NewController` treats a string as a logical view name, while `mvc.NewRestController` writes ordinary values through message converters.
Routes built with `mvc.ResponseBody` always write ordinary values through message converters, even when the containing controller is a normal `mvc.NewController`.
`mvc.Model` infers the logical view name from the request path, while `mvc.ModelAndView` carries an explicit view name, model attributes, and optional view status.
Routes can use `mvc.ResponseStatus` to set the method-level default HTTP status while preserving explicit `ResponseEntity`, redirect, download, and stream result status values.
Routes can use `mvc.RequestMapping` when the same handler should accept the standard MVC HTTP method set, or `mvc.RequestMappingMethods` when the method set must be explicit.
Routes and controllers can use `mvc.WithCrossOrigin` for Spring-style local CORS mappings; the starter keeps the generated preflight route inside the Arkhos deployment.
Routes can use explicit validation groups for request body, multipart body, and model attribute binding.
Multipart routes can bind structured JSON request parts with `mvc.RequestPartJSON[T]` or `mvc.ValidatedRequestPartJSON[T]` while file parts continue to use `mvc.RequestPart`.
Handlers can bind full request parameter and header views through `mvc.RequestParamMap`, `mvc.RequestParamValuesMap`, `mvc.RequestHeaderMap`, and `mvc.RequestHeaderValuesMap`.
Global exception handlers can be packaged as `mvc.NewControllerAdvice` or `mvc.NewRestControllerAdvice`; REST advice writes ordinary exception return values as response bodies by default.
Advice handlers may also return `ResponseEntity[T]` when status, headers, and body must be controlled together.
Routes can call `goark.dev/goark/web.CheckNotModified` before building an expensive body to honor `If-None-Match` and `If-Modified-Since` requests.
Handlers can return `goark.dev/goark/web.NewStatusError` to produce Spring-style status errors through the default Problem Details mapper.
Applications may also register `gbcweb.HTTPClientBuilderCustomizer` beans to add default outbound headers, cookies, interceptors, codecs, or transports to the shared `goark.dev/goark/web/client.Builder`.
The character encoding filter is on by default with UTF-8 request encoding; response encoding can be forced through configuration.
The hidden HTTP method filter is off by default; when enabled it accepts POST form `_method` overrides for PUT, PATCH, and DELETE.
The form content filter is on by default; it exposes PUT, PATCH, and DELETE `application/x-www-form-urlencoded` bodies to MVC `RequestParam` and `ModelAttribute` binding while preserving the request body for later readers.
Applications that need bidirectional WebSocket endpoints can register them with `gbcweb.RegisterWebSocketEndpoint`; the starter contributes the endpoint as a Goark Web configurer and the default Arkhos deployment handles the Servlet upgrade.

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
| `goark.web.resources.static.chain.fixed.version` | unset | Enable a fixed version prefix for static resource paths. |
| `goark.web.mvc.view.enabled` | `true` | Enable default MVC template view resolver when templates exist. |
| `goark.web.mvc.view.templates.location` | `resource/templates` | Template root directory. `classpath:/templates/` maps to `resource/templates`. |
| `goark.web.mvc.view.prefix` | unset | Prefix added to logical view names. |
| `goark.web.mvc.view.suffix` | `.html` | Suffix added to logical view names. |
| `goark.web.mvc.view.content-type` | `text/html; charset=utf-8` | Default template response content type. |
| `goark.web.servlet.encoding.enabled` | `true` | Enable the character encoding filter. |
| `goark.web.servlet.encoding.charset` | `UTF-8` | Default request and response charset. |
| `goark.web.servlet.encoding.force` | unset | Force both request and response charset when set. |
| `goark.web.servlet.encoding.force-request` | `true` | Force request charset when missing or different. |
| `goark.web.servlet.encoding.force-response` | `false` | Force response charset when `Content-Type` is present. |
| `goark.web.filters.hidden-method.enabled` | `false` | Enable POST form `_method` overrides for PUT, PATCH, and DELETE. |
| `goark.web.filters.form-content.enabled` | `true` | Enable PUT, PATCH, and DELETE URL-encoded form body parameters. |
| `goark.web.filters.form-content.max-body-bytes` | `1048576` | Maximum request body cached by the form content filter. |
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
| `spring.web.resources.chain.strategy.fixed.version` | `goark.web.resources.static.chain.fixed.version` |
| `spring.mvc.view.prefix` | `goark.web.mvc.view.prefix` |
| `spring.mvc.view.suffix` | `goark.web.mvc.view.suffix` |
| `spring.servlet.encoding.enabled` | `goark.web.servlet.encoding.enabled` |
| `spring.servlet.encoding.charset` | `goark.web.servlet.encoding.charset` |
| `spring.servlet.encoding.force` | `goark.web.servlet.encoding.force` |
| `spring.servlet.encoding.force-request` | `goark.web.servlet.encoding.force-request` |
| `spring.servlet.encoding.force-response` | `goark.web.servlet.encoding.force-response` |
| `server.servlet.encoding.enabled` | `goark.web.servlet.encoding.enabled` |
| `server.servlet.encoding.charset` | `goark.web.servlet.encoding.charset` |
| `server.servlet.encoding.force` | `goark.web.servlet.encoding.force` |
| `server.servlet.encoding.force-request` | `goark.web.servlet.encoding.force-request` |
| `server.servlet.encoding.force-response` | `goark.web.servlet.encoding.force-response` |
| `spring.mvc.hiddenmethod.filter.enabled` | `goark.web.filters.hidden-method.enabled` |
| `spring.mvc.formcontent.filter.enabled` | `goark.web.filters.form-content.enabled` |

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
- 支持通过 `RequestEntity[T]` 绑定 Spring 风格请求实体，包含请求体、请求头、方法、URL 和 Content-Length 元数据。
- 支持基于 ETag 和 `Last-Modified` 的条件请求，命中时返回 304。
- 支持带路径作用域的 Web 拦截器和 Servlet 过滤器，对齐 Spring MVC 风格 include/exclude 请求链。
- 支持 MVC `Controller` 和 `RestController` 默认返回值语义：普通控制器默认视图，REST 控制器默认响应体。
- 支持 MVC `ResponseBody` 路由包装器，用于普通控制器方法显式将返回值写入响应体。
- 支持 MVC `Model` 和 `ModelAndView` 返回值，用于对齐 Spring 风格服务端模板控制器。
- 支持 MVC `ResponseStatus` 方法级默认状态码，可作用于普通返回值和无响应体处理器。
- 支持未限定 HTTP method 的 MVC `RequestMapping` 路由，并展开为标准 MVC 请求方法集合。
- 支持 MVC 请求体、multipart 请求体和模型属性绑定的显式校验分组。
- 支持 MVC 类型化 JSON request part 绑定和校验分组，用于对齐 Spring 风格 multipart 元数据加文件上传请求。
- 支持 MVC 请求参数和请求头整表绑定，对齐 Spring 风格 `Map` 和多值 map 处理器参数。
- 支持 MVC `ControllerAdvice` 和 `RestControllerAdvice` 异常处理，并复用相同默认返回值语义。
- 默认启用 UTF-8 字符集过滤器，并支持 Spring Boot 兼容的 Servlet 编码属性。
- 支持可选隐藏 HTTP 方法过滤器，用于 HTML 表单方法覆盖。
- 默认启用表单内容过滤器，让 PUT、PATCH 和 DELETE 的 URL 编码请求体可参与参数绑定。
- 支持基于 Arkarta Servlet Upgrade 和默认 Arkhos 嵌入式容器的 WebSocket Endpoint 注册。
- 支持 Goark MVC 请求映射条件，包括 consumes/produces 以及 JSON 兼容厂商媒体类型协商。
- 提供默认出站 `goark.dev/goark/web/client` Builder 与 Client Bean，对齐 Spring 风格 JSON、表单和 multipart Web 客户端用法。
- 提供有序 `HTTPClientBuilderCustomizer` 扩展点，支持业务和其他 starter 定制出站客户端默认行为。

## 使用方式

业务包只需要贡献 MVC 控制器、响应增强器、错误映射器或其他 `goark.dev/goark/web.Configurer` Bean。
本启动器会构建默认 Arkarta Servlet 部署，并通过 Boot 生命周期启动 Arkhos。
MVC 路由可以返回 `goark.dev/goark/web.ResponseEntity[T]`、`Download`、`TextStream`、`BinaryStream`、`SSE` 或 `Redirect`；生成路由和本启动器部署链路会保留响应状态码、响应头、Cookie、缓存头、条件请求校验器、JSON 响应体、协商后的 `produces` 媒体类型、重定向 Location 和流式响应头。
处理器可以绑定 `goark.dev/goark/web.RequestEntity[T]`，在转换后的请求体之外同时读取请求头、方法、完整 URL、路径和 Content-Length 元数据。
通过 `mvc.Return` 构造的路由会遵循所在控制器：`mvc.NewController` 将字符串作为逻辑视图名，`mvc.NewRestController` 将普通返回值通过消息转换器写入响应体。
通过 `mvc.ResponseBody` 构造的路由始终把普通返回值通过消息转换器写入响应体，即使所在控制器是普通 `mvc.NewController`。
`mvc.Model` 会根据请求路径推导逻辑视图名，`mvc.ModelAndView` 则携带显式视图名、模型属性和可选视图状态码。
路由可以通过 `mvc.ResponseStatus` 设置方法级默认 HTTP 状态码，同时保留 `ResponseEntity`、重定向、下载和流式结果自身显式声明的状态码。
路由可以通过 `mvc.RequestMapping` 让同一个处理器接收标准 MVC HTTP 方法集合，也可以通过 `mvc.RequestMappingMethods` 显式指定方法集合。
路由可以对请求体、multipart 请求体和模型属性绑定使用显式校验分组。
multipart 路由可以通过 `mvc.RequestPartJSON[T]` 或 `mvc.ValidatedRequestPartJSON[T]` 绑定结构化 JSON request part，文件段继续使用 `mvc.RequestPart`。
处理器可以通过 `mvc.RequestParamMap`、`mvc.RequestParamValuesMap`、`mvc.RequestHeaderMap` 和 `mvc.RequestHeaderValuesMap` 绑定完整请求参数和请求头视图。
全局异常处理器可以组织为 `mvc.NewControllerAdvice` 或 `mvc.NewRestControllerAdvice`；REST advice 默认把普通异常返回值写为响应体。
Advice 处理器也可以返回 `ResponseEntity[T]`，用于同时控制状态码、响应头和响应体。
路由可以在构造高成本响应体前调用 `goark.dev/goark/web.CheckNotModified`，以处理 `If-None-Match` 和 `If-Modified-Since` 请求。
处理器可以返回 `goark.dev/goark/web.NewStatusError`，由默认 Problem Details 映射器输出对齐 Spring 风格的状态错误。
业务也可以注册 `gbcweb.HTTPClientBuilderCustomizer` Bean，为共享 `goark.dev/goark/web/client.Builder` 添加默认出站请求头、Cookie、拦截器、编解码器或底层传输。
字符集过滤器默认启用，默认请求字符集为 UTF-8；响应字符集是否强制覆盖可通过配置控制。
隐藏 HTTP 方法过滤器默认关闭；启用后接受 POST 表单 `_method` 覆盖 PUT、PATCH 和 DELETE。
表单内容过滤器默认启用；它会把 PUT、PATCH 和 DELETE 的 `application/x-www-form-urlencoded` 请求体暴露给 MVC `RequestParam` 和 `ModelAttribute` 绑定，并保留请求体供后续读取。
需要双向通信的业务可以通过 `gbcweb.RegisterWebSocketEndpoint` 注册 WebSocket 端点；本启动器会将端点贡献为 Goark Web 配置器，并由默认 Arkhos 部署处理 Servlet Upgrade。

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
| `goark.web.resources.static.chain.fixed.version` | 未设置 | 静态资源固定版本路径前缀。 |
| `goark.web.mvc.view.enabled` | `true` | 模板存在时启用默认 MVC 模板视图解析器。 |
| `goark.web.mvc.view.templates.location` | `resource/templates` | 模板根目录，`classpath:/templates/` 会映射为 `resource/templates`。 |
| `goark.web.mvc.view.prefix` | 未设置 | 逻辑视图名前缀。 |
| `goark.web.mvc.view.suffix` | `.html` | 逻辑视图名后缀。 |
| `goark.web.mvc.view.content-type` | `text/html; charset=utf-8` | 模板响应默认媒体类型。 |
| `goark.web.servlet.encoding.enabled` | `true` | 是否启用字符集过滤器。 |
| `goark.web.servlet.encoding.charset` | `UTF-8` | 默认请求和响应字符集。 |
| `goark.web.servlet.encoding.force` | 未设置 | 设置后同时强制请求和响应字符集。 |
| `goark.web.servlet.encoding.force-request` | `true` | 是否强制请求字符集。 |
| `goark.web.servlet.encoding.force-response` | `false` | 响应存在 `Content-Type` 时是否强制响应字符集。 |
| `goark.web.filters.hidden-method.enabled` | `false` | 是否启用 POST 表单 `_method` 覆盖，默认允许 PUT、PATCH 和 DELETE。 |
| `goark.web.filters.form-content.enabled` | `true` | 是否启用 PUT、PATCH 和 DELETE 的 URL 编码表单体参数绑定。 |
| `goark.web.filters.form-content.max-body-bytes` | `1048576` | 表单内容过滤器最大缓存请求体字节数。 |
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
| `spring.web.resources.chain.strategy.fixed.version` | `goark.web.resources.static.chain.fixed.version` |
| `spring.mvc.view.prefix` | `goark.web.mvc.view.prefix` |
| `spring.mvc.view.suffix` | `goark.web.mvc.view.suffix` |
| `spring.servlet.encoding.enabled` | `goark.web.servlet.encoding.enabled` |
| `spring.servlet.encoding.charset` | `goark.web.servlet.encoding.charset` |
| `spring.servlet.encoding.force` | `goark.web.servlet.encoding.force` |
| `spring.servlet.encoding.force-request` | `goark.web.servlet.encoding.force-request` |
| `spring.servlet.encoding.force-response` | `goark.web.servlet.encoding.force-response` |
| `server.servlet.encoding.enabled` | `goark.web.servlet.encoding.enabled` |
| `server.servlet.encoding.charset` | `goark.web.servlet.encoding.charset` |
| `server.servlet.encoding.force` | `goark.web.servlet.encoding.force` |
| `server.servlet.encoding.force-request` | `goark.web.servlet.encoding.force-request` |
| `server.servlet.encoding.force-response` | `goark.web.servlet.encoding.force-response` |
| `spring.mvc.hiddenmethod.filter.enabled` | `goark.web.filters.hidden-method.enabled` |
| `spring.mvc.formcontent.filter.enabled` | `goark.web.filters.form-content.enabled` |

服务端监听、multipart 和 Servlet async 超时属性由默认包含的 `goark.dev/gbc-arkhos` 提供。

## 开发

```bash
go test ./...
```
