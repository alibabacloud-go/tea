package dara

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

func Test_ReadAsBytes(t *testing.T) {
	byt, err := ReadAsBytes(strings.NewReader("common"))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", string(byt))

	byt, err = ReadAsBytes(ioutil.NopCloser(strings.NewReader("common")))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", string(byt))
}

func Test_ReadAsJSON(t *testing.T) {
	result, err := ReadAsJSON(strings.NewReader(`{"cleint":"test"}`))
	if res, ok := result.(map[string]interface{}); ok {
		utils.AssertNil(t, err)
		utils.AssertEqual(t, "test", res["cleint"])
	}

	result, err = ReadAsJSON(strings.NewReader(""))
	utils.AssertNil(t, err)
	utils.AssertNil(t, result)

	result, err = ReadAsJSON(ioutil.NopCloser(strings.NewReader(`{"cleint":"test"}`)))
	if res, ok := result.(map[string]interface{}); ok {
		utils.AssertNil(t, err)
		utils.AssertEqual(t, "test", res["cleint"])
	}
}

func Test_ReadAsString(t *testing.T) {
	str, err := ReadAsString(strings.NewReader("common"))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", str)

	str, err = ReadAsString(ioutil.NopCloser(strings.NewReader("common")))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", str)
}
