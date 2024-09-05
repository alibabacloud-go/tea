package utils

import (
	"net/http"
	"testing"
)

func Test_isNil(t *testing.T) {
	var req *http.Request
	isnil := isNil(nil)
	AssertEqual(t, true, isnil)
	isnil = isNil(req)
	AssertEqual(t, true, isnil)
	AssertNil(t, nil)
	AssertContains(t, "tea test", "test")
	AssertNotNil(t, "sdk")
}
