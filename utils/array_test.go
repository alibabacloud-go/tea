package utils

import (
	"testing"
)

func Test_Contains(t *testing.T) {
	apple := "apple"
	banana := "banana"
	cherry := "cherry"
	slice := []*string{&apple, &banana, &cherry, nil}
	AssertEqual(t, true, Contains(slice, &banana))
	toFind := "banana"
	AssertEqual(t, true, Contains(slice, &toFind))
	notFind := "notFind"
	AssertEqual(t, false, Contains(slice, &notFind))
	notFind = ""
	AssertEqual(t, false, Contains(slice, &notFind))
	AssertEqual(t, false, Contains(slice, nil))
	AssertEqual(t, false, Contains(nil, nil))
}
