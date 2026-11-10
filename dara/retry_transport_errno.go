//go:build !plan9
// +build !plan9

package dara

import (
	"errors"
	"runtime"
	"syscall"
)

func transportErrnoCode(err error) string {
	switch {
	case errors.Is(err, syscall.ECONNRESET):
		return "ECONNRESET"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "ECONNREFUSED"
	case errors.Is(err, syscall.ECONNABORTED):
		return "ECONNABORTED"
	case errors.Is(err, syscall.EPIPE):
		return "EPIPE"
	}
	// Windows 的 socket 错误使用 WSA 错误码，而非上面的 POSIX 常量。
	if runtime.GOOS == "windows" {
		switch {
		case errors.Is(err, syscall.Errno(10054)):
			return "ECONNRESET"
		case errors.Is(err, syscall.Errno(10061)):
			return "ECONNREFUSED"
		case errors.Is(err, syscall.Errno(10053)):
			return "ECONNABORTED"
		}
	}
	return ""
}
