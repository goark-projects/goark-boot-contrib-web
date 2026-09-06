package gbcweb_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	arkjson "goark.dev/arkarta/json"
	servletmultipart "goark.dev/arkarta/servlet/multipart"
	arkweb "goark.dev/arkarta/web"
	"goark.dev/arkarta/websocket/frame"
	"goark.dev/boot"
	"goark.dev/boot/configdata"
	gbcarkhos "goark.dev/gbc-arkhos"
	gbcweb "goark.dev/gbc-web"
	"goark.dev/goark"
	"goark.dev/goark/container"
	"goark.dev/goark/web/mvc"
)

func TestAutoConfigure_whenConvertersExist_shouldBindMVCRequestParameters(t *testing.T) {
	app, err := boot.Run(
		t.Context(),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure(
			gbcweb.WithArkhosOptions(gbcarkhos.WithAddress("127.0.0.1:0")),
		)),
		boot.WithConfiguration(
			starterConversionConfiguration{},
			mvc.NewConfiguration("test.mvc.conversion", mvc.NewController(
				"conversion",
				mvc.GET(
					"/conversion",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterConversionPayload, error) {
							page, err := mvc.RequestParamInt(ctx, "page")
							if err != nil {
								return starterConversionPayload{}, err
							}
							tenant, err := mvc.RequestParamAs[starterTenantID](ctx, "tenant")
							if err != nil {
								return starterConversionPayload{}, err
							}
							return starterConversionPayload{
								Page:   page,
								Tenant: tenant.value,
							}, nil
						},
					),
				),
				mvc.GET(
					"/search",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterSearchPayload, error) {
							criteria, err := mvc.ModelAttribute[starterSearchCriteria](ctx)
							if err != nil {
								return starterSearchPayload{}, err
							}
							return starterSearchPayload{
								Page:   criteria.Page,
								Tenant: criteria.Tenant.value,
							}, nil
						},
					),
				),
				mvc.GET(
					"/search/nested",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterNestedSearchPayload, error) {
							criteria, err := mvc.ModelAttribute[starterNestedSearchCriteria](ctx)
							if err != nil {
								return starterNestedSearchPayload{}, err
							}
							return starterNestedSearchPayload{
								OwnerName:  criteria.Owner.Name,
								OwnerLevel: criteria.Owner.Level,
								Page:       criteria.Page,
							}, nil
						},
					),
				),
				mvc.GET(
					"/search/indexed",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterIndexedSearchPayload, error) {
							criteria, err := mvc.ModelAttribute[starterIndexedSearchCriteria](ctx)
							if err != nil {
								return starterIndexedSearchPayload{}, err
							}
							return starterIndexedSearchPayload{
								FirstName:   criteria.Owners[0].Name,
								FirstLevel:  criteria.Owners[0].Level,
								FirstAlias:  criteria.Owners[0].Aliases[0],
								SecondName:  criteria.Owners[1].Name,
								SecondLevel: criteria.Owners[1].Level,
								SecondAlias: criteria.Owners[1].Aliases[0],
								Page:        criteria.Page,
							}, nil
						},
					),
				),
				mvc.GET(
					"/search/mapped",
					mvc.JSON(
						http.StatusOK,
						func(ctx *arkweb.Context) (starterMappedSearchPayload, error) {
							criteria, err := mvc.ModelAttribute[starterMappedSearchCriteria](ctx)
							if err != nil {
								return starterMappedSearchPayload{}, err
							}
							return starterMappedSearchPayload{
								Level:  criteria.Filters["level"],
								Roles:  criteria.Tags["roles"],
								Groups: criteria.Tags["groups"],
							}, nil
						},
					),
				),
			)),
		),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	snapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/conversion?page=abcd&tenant=blue",
		http.StatusOK,
	)
	var payload starterConversionPayload
	if err := arkjson.Unmarshal(nil, []byte(snapshot.body), &payload); err != nil {
		t.Fatalf("conversion json invalid: %v", err)
	}
	if payload.Page != 104 || payload.Tenant != "tenant:blue" {
		t.Fatalf("payload = %#v, want converted parameters", payload)
	}

	searchSnapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/search?page=abcde&tenant=green",
		http.StatusOK,
	)
	var searchPayload starterSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(searchSnapshot.body), &searchPayload); err != nil {
		t.Fatalf("search json invalid: %v", err)
	}
	if searchPayload.Page != 105 || searchPayload.Tenant != "tenant:green" {
		t.Fatalf("search payload = %#v, want converted model attribute", searchPayload)
	}

	nestedSnapshot := requestUntilStatusSnapshot(
		t,
		starterServerURL(t, app)+"/search/nested?owner.name=ada&owner.level=admin&page=xy",
		http.StatusOK,
	)
	var nestedPayload starterNestedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(nestedSnapshot.body), &nestedPayload); err != nil {
		t.Fatalf("nested search json invalid: %v", err)
	}
	if nestedPayload.OwnerName != "ada" || nestedPayload.OwnerLevel != 105 ||
		nestedPayload.Page != 102 {
		t.Fatalf("nested search payload = %#v, want nested model attribute", nestedPayload)
	}

	indexedSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search/indexed?"+
		"owners[0].name=ada&owners[0].level=admin&owners[0].aliases[0]=lead&"+
		"owners[1].name=linus&owners[1].level=kernel&"+
		"owners[1].aliases[0]=maintainer&page=xy",
		http.StatusOK,
	)
	var indexedPayload starterIndexedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(indexedSnapshot.body), &indexedPayload); err != nil {
		t.Fatalf("indexed search json invalid: %v", err)
	}
	if indexedPayload.FirstName != "ada" ||
		indexedPayload.FirstLevel != 105 ||
		indexedPayload.FirstAlias != "lead" ||
		indexedPayload.SecondName != "linus" ||
		indexedPayload.SecondLevel != 106 ||
		indexedPayload.SecondAlias != "maintainer" ||
		indexedPayload.Page != 102 {
		t.Fatalf("indexed search payload = %#v, want indexed model attribute", indexedPayload)
	}

	mappedSnapshot := requestUntilStatusSnapshot(t, starterServerURL(t, app)+"/search/mapped?"+
		"filters[level]=admin&tags[roles]=admin,ops&tags[groups]=core&tags[groups]=web", http.StatusOK)
	var mappedPayload starterMappedSearchPayload
	if err := arkjson.Unmarshal(nil, []byte(mappedSnapshot.body), &mappedPayload); err != nil {
		t.Fatalf("mapped search json invalid: %v", err)
	}
	if mappedPayload.Level != 105 ||
		len(mappedPayload.Roles) != 2 ||
		mappedPayload.Roles[0] != "admin" ||
		mappedPayload.Roles[1] != "ops" ||
		len(mappedPayload.Groups) != 2 ||
		mappedPayload.Groups[0] != "core" ||
		mappedPayload.Groups[1] != "web" {
		t.Fatalf("mapped search payload = %#v, want mapped model attribute", mappedPayload)
	}
}
func TestAutoConfigure_whenControllerRequestMethodsExist_shouldServeCombinedMethodsThroughArkhos(
	t *testing.T,
) {
	root := t.TempDir()
	writeFile(t, root+"/app.yml", `
server:
  address: 127.0.0.1
  port: 0
goark:
  web:
`)

	handler := mvc.JSON(
		http.StatusOK,
		func(ctx *arkweb.Context) (starterRequestMethodPayload, error) {
			return starterRequestMethodPayload{Method: ctx.Request().Method()}, nil
		},
	)
	routes := mvc.RequestMapping("/controller-methods/implicit", handler)
	routes = append(routes, mvc.GET("/controller-methods/explicit", handler))
	controller := mvc.NewRestController("methods", routes...).
		WithRequestMethods(http.MethodPost, http.MethodTrace)

	app, err := boot.Run(
		t.Context(),
		boot.WithConfigDataOptions(configdata.WithLocations(root)),
		boot.WithAutoConfiguration(gbcweb.AutoConfigure()),
		boot.WithConfiguration(mvc.NewConfiguration("test.mvc.request-methods", controller)),
	)
	if err != nil {
		t.Fatalf("boot run failed: %v", err)
	}
	defer closeApp(t, app)

	serverURL := starterServerURL(t, app)
	for _, method := range []string{http.MethodPost, http.MethodTrace} {
		assertRequestMethodRoute(t, method, serverURL+"/controller-methods/implicit")
	}
	requestUntilStatusWith(t, func() (*http.Request, error) {
		return http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			serverURL+"/controller-methods/implicit",
			nil,
		)
	}, http.StatusMethodNotAllowed)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodTrace} {
		assertRequestMethodRoute(t, method, serverURL+"/controller-methods/explicit")
	}
}

func dialStarterWebSocket(
	t *testing.T,
	rawURL string,
	protocol string,
) (net.Conn, *bufio.Reader, http.Header) {
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
	requestFormat := "GET %s HTTP/1.1\r\nHost: %s\r\n" +
		"Connection: Upgrade\r\nUpgrade: websocket\r\n" +
		"Sec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n" +
		"Sec-WebSocket-Protocol: %s\r\n\r\n"
	if _, err := fmt.Fprintf(
		conn, requestFormat,
		parsed.RequestURI(), parsed.Host, starterWebSocketKey, protocol,
	); err != nil {
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

func assertStarterPreferencePayload(t *testing.T, payload starterPreferencePayload) {
	t.Helper()
	if payload.Theme != "dark" ||
		!payload.NotifySet ||
		payload.Notify ||
		!payload.ConfirmSet ||
		!payload.Confirm ||
		!payload.ProfileSet ||
		payload.Subscribed ||
		payload.TagsNil ||
		payload.TagsLength != 0 {
		t.Fatalf("payload = %#v, want field prefix binding", payload)
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

func (c starterWebSocketConfiguration) Register(
	ctx context.Context,
	registry *container.Registry,
) error {
	return c.RegisterWithContext(ctx, goark.NewConfigurationContext(nil, registry))
}

type starterGroupedCreateRequest struct {
	Name string `json:"name" arkarta:"required" arkarta-groups:"create"`
	Code string `json:"code" arkarta:"required"`
}

type starterBindingUploadRequest struct {
	Title string                `form:"title" arkarta:"required"`
	File  servletmultipart.Part `                                multipart:"file"`
}

type starterPreferenceProfile struct {
	Subscribed bool `form:"subscribed" json:"subscribed"`
}

type starterModelNameAccount struct {
	Name string
}

type starterHTTPClientCustomizerConfiguration struct{}
