package dara

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/utils"
	"github.com/gorilla/websocket"
)

// listenLoopback starts an HTTP server bound only to the given loopback address
// ("127.0.0.1" or "::1"); it skips the test when the address family is unavailable.
func listenLoopback(t *testing.T, ip string) (*httptest.Server, int) {
	l, err := net.Listen("tcp", net.JoinHostPort(ip, "0"))
	if err != nil {
		t.Skipf("loopback %s unavailable: %v", ip, err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	server.Listener.Close()
	server.Listener = l
	server.Start()
	return server, l.Addr().(*net.TCPAddr).Port
}

// requireDualStackLocalhost skips the test unless "localhost" resolves to both
// an A record and an AAAA record.
func requireDualStackLocalhost(t *testing.T) {
	ips, err := net.LookupIP("localhost")
	if err != nil {
		t.Skipf("lookup localhost failed: %v", err)
	}
	var hasV4, hasV6 bool
	for _, ip := range ips {
		if ip.To4() != nil {
			hasV4 = true
		} else {
			hasV6 = true
		}
	}
	if !hasV4 || !hasV6 {
		t.Skipf("localhost is not dual-stack: %v", ips)
	}
}

func Test_getDialNetwork(t *testing.T) {
	utils.AssertEqual(t, "tcp", getDialNetwork("tcp", false))
	utils.AssertEqual(t, "tcp6", getDialNetwork("tcp6", false))
	utils.AssertEqual(t, "", getDialNetwork("", false))
	utils.AssertEqual(t, "tcp4", getDialNetwork("tcp", true))
	utils.AssertEqual(t, "tcp4", getDialNetwork("tcp4", true))
	utils.AssertEqual(t, "tcp4", getDialNetwork("tcp6", true))
	utils.AssertEqual(t, "tcp4", getDialNetwork("", true))
	utils.AssertEqual(t, "udp", getDialNetwork("udp", true))
}

func Test_NewRuntimeObject_Ipv4Only(t *testing.T) {
	utils.AssertNil(t, NewRuntimeObject(map[string]interface{}{}).Ipv4Only)
	utils.AssertEqual(t, true, BoolValue(NewRuntimeObject(map[string]interface{}{"ipv4Only": true}).Ipv4Only))
	utils.AssertEqual(t, false, BoolValue(NewRuntimeObject(map[string]interface{}{"ipv4Only": false}).Ipv4Only))
}

func Test_getClientTag_Ipv4Only(t *testing.T) {
	dual := &RuntimeObject{}
	v4 := &RuntimeObject{Ipv4Only: Bool(true)}
	utils.AssertEqual(t, false, dual.getClientTag("ecs.aliyuncs.com") == v4.getClientTag("ecs.aliyuncs.com"))
	utils.AssertEqual(t, dual.getClientTag("ecs.aliyuncs.com"), (&RuntimeObject{Ipv4Only: Bool(false)}).getClientTag("ecs.aliyuncs.com"))
}

func Test_setDialContext_Ipv4Only(t *testing.T) {
	ln, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	addr := ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := setDialContext(&RuntimeObject{})(ctx, "tcp", addr)
	utils.AssertNil(t, err)
	conn.Close()

	conn, err = setDialContext(&RuntimeObject{Ipv4Only: Bool(true)})(ctx, "tcp", addr)
	utils.AssertNil(t, conn)
	utils.AssertNotNil(t, err)
	utils.AssertContains(t, err.Error(), "tcp4")
}

func Test_DoRequest_Ipv4Only(t *testing.T) {
	requireDualStackLocalhost(t)

	v6Server, v6Port := listenLoopback(t, "::1")
	defer v6Server.Close()
	v4Server, v4Port := listenLoopback(t, "127.0.0.1")
	defer v4Server.Close()

	newRequest := func(port int) *Request {
		req := NewRequest()
		req.Protocol = String("http")
		req.Method = String("GET")
		req.Headers["host"] = String("localhost")
		req.Port = Int(port)
		return req
	}

	// Dual-stack (default) reaches an AAAA-only target and keeps the connection pooled.
	res, err := DoRequest(newRequest(v6Port), NewRuntimeObject(map[string]interface{}{"connectTimeout": 2000}))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, 200, IntValue(res.StatusCode))
	res.ReadBody()

	// ipv4Only must not reuse the pooled IPv6 connection nor fall back to IPv6.
	ipv4Runtime := NewRuntimeObject(map[string]interface{}{"connectTimeout": 2000, "ipv4Only": true})
	res, err = DoRequest(newRequest(v6Port), ipv4Runtime)
	utils.AssertNil(t, res)
	utils.AssertNotNil(t, err)

	// ipv4Only still reaches a target listening on IPv4.
	res, err = DoRequest(newRequest(v4Port), ipv4Runtime)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, 200, IntValue(res.StatusCode))
	res.ReadBody()

	// Other clients in the same process are unaffected and still use IPv6.
	res, err = DoRequest(newRequest(v6Port), NewRuntimeObject(map[string]interface{}{"connectTimeout": 2000, "ipv4Only": false}))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, 200, IntValue(res.StatusCode))
	res.ReadBody()
}

func Test_getHttpTransport_Socks5Ipv4Only(t *testing.T) {
	ln, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer ln.Close()
	req := &Request{Protocol: String("http"), Domain: String("ecs.aliyuncs.com"), Headers: map[string]*string{}}
	runtime := &RuntimeObject{
		Socks5Proxy:   String("socks5://" + ln.Addr().String()),
		Socks5NetWork: String("tcp"),
		Ipv4Only:      Bool(true),
	}
	trans, err := getHttpTransport(req, runtime)
	utils.AssertNil(t, err)
	_, err = trans.Dial("tcp", "ecs.aliyuncs.com:80")
	utils.AssertNotNil(t, err)
	utils.AssertContains(t, err.Error(), "tcp4")
}

func Test_WebSocketDialer_Ipv4Only(t *testing.T) {
	ln, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	addr := ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c := &DefaultWebSocketClient{}

	dialer := &websocket.Dialer{}
	c.configureNetDialer(dialer, &RuntimeObject{}, time.Second)
	conn, err := dialer.NetDialContext(ctx, "tcp", addr)
	utils.AssertNil(t, err)
	conn.Close()

	dialer = &websocket.Dialer{}
	c.configureNetDialer(dialer, &RuntimeObject{Ipv4Only: Bool(true)}, time.Second)
	_, err = dialer.NetDialContext(ctx, "tcp", addr)
	utils.AssertNotNil(t, err)
	utils.AssertContains(t, err.Error(), "tcp4")

	dialer = &websocket.Dialer{}
	err = c.configureSOCKS5Proxy(dialer, &RuntimeObject{
		Socks5Proxy: String("socks5://" + addr),
		Ipv4Only:    Bool(true),
	}, time.Second)
	utils.AssertNil(t, err)
	_, err = dialer.NetDialContext(ctx, "tcp", "ecs.aliyuncs.com:"+strconv.Itoa(443))
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, true, strings.Contains(err.Error(), "tcp4"))
}
