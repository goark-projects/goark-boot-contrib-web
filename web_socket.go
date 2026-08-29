package gbcweb

import (
	"goark.dev/goark/container"
	gowebsocket "goark.dev/goark/web/websocket"
)

// WebSocketEndpoint 处理一个 WebSocket 会话的生命周期与消息。
type WebSocketEndpoint = gowebsocket.Endpoint

// WebSocketEndpointFunc 将函数组适配为 WebSocketEndpoint。
type WebSocketEndpointFunc = gowebsocket.EndpointFunc

// WebSocketSession 表示一个 WebSocket 会话。
type WebSocketSession = gowebsocket.Session

// WebSocketHandshake 表示一次成功的 WebSocket 握手。
type WebSocketHandshake = gowebsocket.Handshake

// WebSocketOption 定制 WebSocket 端点注册。
type WebSocketOption = gowebsocket.Option

// WebSocketHandshakeOption 定制 WebSocket 握手协商。
type WebSocketHandshakeOption = gowebsocket.HandshakeOption

// WebSocketFrameConnectionOption 定制 WebSocket 帧连接。
type WebSocketFrameConnectionOption = gowebsocket.FrameConnectionOption

// WebSocketSessionIDGenerator 为每个 WebSocket 连接生成会话标识。
type WebSocketSessionIDGenerator = gowebsocket.SessionIDGenerator

// RegisterWebSocketEndpoint 注册 WebSocket Endpoint 配置器 Bean。
func RegisterWebSocketEndpoint(registry *container.Registry, name string, pattern string, endpoint WebSocketEndpoint, options ...WebSocketOption) error {
	return gowebsocket.RegisterEndpoint(registry, name, pattern, endpoint, options...)
}

// WithWebSocketServletName 设置底层 Servlet 名称。
func WithWebSocketServletName(name string) WebSocketOption {
	return gowebsocket.WithServletName(name)
}

// WithWebSocketSubprotocols 设置服务端支持的 WebSocket 子协议。
func WithWebSocketSubprotocols(protocols ...string) WebSocketOption {
	return gowebsocket.WithSubprotocols(protocols...)
}

// WithWebSocketHandshakeOptions 追加底层 WebSocket 握手选项。
func WithWebSocketHandshakeOptions(options ...WebSocketHandshakeOption) WebSocketOption {
	return gowebsocket.WithHandshakeOptions(options...)
}

// WithWebSocketFrameOptions 追加底层 WebSocket 帧连接选项。
func WithWebSocketFrameOptions(options ...WebSocketFrameConnectionOption) WebSocketOption {
	return gowebsocket.WithFrameOptions(options...)
}

// WithWebSocketMaxFrameBytes 设置单帧最大载荷字节数。
func WithWebSocketMaxFrameBytes(maxBytes int64) WebSocketOption {
	return gowebsocket.WithMaxFrameBytes(maxBytes)
}

// WithWebSocketMaxMessageBytes 设置聚合消息最大载荷字节数。
func WithWebSocketMaxMessageBytes(maxBytes int64) WebSocketOption {
	return gowebsocket.WithMaxMessageBytes(maxBytes)
}

// WithWebSocketSessionIDGenerator 设置 WebSocket 会话标识生成器。
func WithWebSocketSessionIDGenerator(generator WebSocketSessionIDGenerator) WebSocketOption {
	return gowebsocket.WithSessionIDGenerator(generator)
}
