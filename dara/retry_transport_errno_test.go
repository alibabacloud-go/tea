//go:build !plan9
// +build !plan9

package dara

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestTransportRetryErrno(t *testing.T) {
	for _, test := range []struct {
		err  syscall.Errno
		code string
	}{
		{syscall.ECONNRESET, "ECONNRESET"},
		{syscall.ECONNREFUSED, "ECONNREFUSED"},
		{syscall.ECONNABORTED, "ECONNABORTED"},
		{syscall.EPIPE, "EPIPE"},
	} {
		t.Run(test.code, func(t *testing.T) {
			opErr := &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", test.err)}
			assertTransportRetry(t, opErr, test.code)
			urlErr := &url.Error{Op: "Post", URL: "http://localhost", Err: opErr}
			assertTransportRetry(t, fmt.Errorf("request: %w", urlErr), test.code)
			if !errors.Is(urlErr, test.err) {
				t.Error("errno must remain accessible in the error chain")
			}
		})
	}
	if runtime.GOOS == "windows" {
		for errno, code := range map[syscall.Errno]string{10053: "ECONNABORTED", 10054: "ECONNRESET", 10061: "ECONNREFUSED"} {
			assertTransportRetry(t, &net.OpError{Op: "read", Net: "tcp", Err: errno}, code)
		}
	}
	options := &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{{Exception: []string{"RetryError"}}}}
	for _, err := range []error{syscall.ECONNRESET, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.EACCES}} {
		if ShouldRetry(options, &RetryPolicyContext{RetriesAttempted: 1, Exception: err}) {
			t.Errorf("unexpected retry for %v", err)
		}
	}
}

func TestTransportRetryHTTPReset(t *testing.T) {
	for _, bodyReset := range []bool{false, true} {
		t.Run(fmt.Sprintf("bodyReset=%v", bodyReset), func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := ln.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(3 * time.Second))
				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err == nil {
					_, err = ioutil.ReadAll(req.Body)
					req.Body.Close()
				}
				if err == nil && bodyReset {
					_, err = fmt.Fprint(conn, "HTTP/1.1 200 OK\r\nContent-Length: 1024\r\n\r\npartial")
					time.Sleep(20 * time.Millisecond)
				}
				if err == nil {
					err = conn.(*net.TCPConn).SetLinger(0)
				}
				done <- err
			}()
			transport := &http.Transport{}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: time.Second}
			resp, err := client.Post("http://"+ln.Addr().String(), "text/plain", nil)
			if err == nil {
				_, err = ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
			if serverErr := <-done; serverErr != nil {
				t.Fatal(serverErr)
			}
			if !errors.Is(err, syscall.ECONNRESET) && !(runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(10054))) {
				t.Fatalf("expected connection reset, got %T (%v)", err, err)
			}
			assertTransportRetry(t, err, "ECONNRESET")
		})
	}
}
