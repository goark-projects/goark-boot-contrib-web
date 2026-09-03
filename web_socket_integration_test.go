package gbcweb_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"time"

	arkws "goark.dev/arkarta/websocket"
	"goark.dev/arkarta/websocket/frame"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	"goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
)

const starterWebSocketKey = "dGhlIHNhbXBsZSBub25jZQ=="

func TestRegisterWebSocketEndpoint_whenEndpointExists_shouldRegisterConfigurerBean(t *testing.T) {
	registry := container.NewRegistry()
	if err := gbcweb.RegisterWebSocketEndpoint(registry, "chatSocket", "/ws/chat", gbcweb.WebSocketEndpointFunc{}); err != nil {
		t.Fatalf("RegisterWebSocketEndpoint failed: %v", err)
	}
	if _, exists := registry.Definition("chatSocket"); !exists {
		t.Fatal("websocket configurer bean should be registered")
	}
}

func TestRegisterWebSocketEndpoint_whenEndpointIsNil_shouldReturnError(t *testing.T) {
	registry := container.NewRegistry()
	err := gbcweb.RegisterWebSocketEndpoint(registry, "badSocket", "/ws/bad", nil)
	if !errors.Is(err, arkws.ErrNilEndpoint) {
		t.Fatalf("err = %v, want ErrNilEndpoint", err)
	}
}

func TestAutoConfigure_whenWebSocketEndpointRegistered_shouldUpgradeAndEcho(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(starterWebSocketConfiguration{}),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	conn, reader, headers := dialStarterWebSocket(t, starterServerURL(t, app)+"/ws/chat", "chat")
	defer conn.Close()

	if headers.Get("Sec-Websocket-Protocol") != "chat" {
		t.Fatalf("Sec-WebSocket-Protocol = %q, want chat", headers.Get("Sec-Websocket-Protocol"))
	}
	writeClientFrame(t, conn, frame.New(frame.OpText, []byte("hello"), frame.WithMask(frame.MaskKey{1, 2, 3, 4})))
	echo := readServerFrame(t, reader)
	if echo.OpCode() != frame.OpText || string(echo.Payload()) != "echo:hello" {
		t.Fatalf("echo frame = %s/%q, want text echo", echo.OpCode(), string(echo.Payload()))
	}
	writeClientFrame(t, conn, frame.New(frame.OpClose, nil, frame.WithMask(frame.MaskKey{4, 3, 2, 1})))
}

type starterWebSocketConfiguration struct{}

func (starterWebSocketConfiguration) Name() string {
	return "test.web.websocket"
}

func (starterWebSocketConfiguration) Order() int {
	return 0
}

func (c starterWebSocketConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

func (starterWebSocketConfiguration) RegisterWithContext(_ context.Context, config goark.ConfigurationContext) error {
	return gbcweb.RegisterWebSocketEndpoint(
		config.Registry(),
		"testChatWebSocket",
		"/ws/chat",
		gbcweb.WebSocketEndpointFunc{
			Text: func(ctx context.Context, session gbcweb.WebSocketSession, text string) error {
				return session.SendText(ctx, "echo:"+text)
			},
		},
		gbcweb.WithWebSocketServletName("testChatSocket"),
		gbcweb.WithWebSocketSubprotocols("chat"),
	)
}

func dialStarterWebSocket(t *testing.T, rawURL string, protocol string) (net.Conn, *bufio.Reader, http.Header) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse websocket url failed: %v", err)
	}
	conn, err := net.DialTimeout("tcp", parsed.Host, 3*time.Second)
	if err != nil {
		t.Fatalf("dial websocket failed: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Protocol: %s\r\n\r\n", parsed.RequestURI(), parsed.Host, starterWebSocketKey, protocol); err != nil {
		t.Fatalf("write handshake failed: %v", err)
	}

	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read handshake status failed: %v", err)
	}
	if !strings.HasPrefix(status, "HTTP/1.1 101 ") && !strings.HasPrefix(status, "HTTP/1.0 101 ") {
		t.Fatalf("handshake status = %q, want 101", strings.TrimSpace(status))
	}
	headers := readHandshakeHeaders(t, reader)
	if headers.Get("Upgrade") != "websocket" || headers.Get("Sec-Websocket-Accept") == "" {
		t.Fatalf("handshake headers = %#v", headers)
	}
	return conn, reader, headers
}

func readHandshakeHeaders(t *testing.T, reader *bufio.Reader) http.Header {
	t.Helper()

	headers := http.Header{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read handshake header failed: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return headers
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			t.Fatalf("invalid handshake header line %q", line)
		}
		headers.Add(textproto.CanonicalMIMEHeaderKey(name), strings.TrimSpace(value))
	}
}

func writeClientFrame(t *testing.T, conn net.Conn, next frame.Frame) {
	t.Helper()
	if err := frame.Write(conn, next); err != nil {
		t.Fatalf("write websocket frame failed: %v", err)
	}
}

func readServerFrame(t *testing.T, reader *bufio.Reader) frame.Frame {
	t.Helper()
	next, err := frame.Read(reader, frame.WithMaskPolicy(frame.MaskForbidden))
	if err != nil {
		t.Fatalf("read websocket frame failed: %v", err)
	}
	return next
}
