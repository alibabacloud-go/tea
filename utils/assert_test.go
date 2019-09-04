package utils

import (
	"net/http"
	"testing"
)

func Test_isNil(t *testing.T) {
	isnil := isNil(nil)
	AssertEqual(t, true, isnil)
	var req *http.Request
	isnil = isNil(req)
	AssertEqual(t, true, isnil)
	AssertNil(t, nil)
	AssertContains(t, "tea test", "test")
	AssertNotNil(t, "sdk")
}
