package dara

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

func Test_ToFormString(t *testing.T) {
	str := ToFormString(nil)
	utils.AssertEqual(t, "", str)

	a := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	str = ToFormString(a)
	utils.AssertEqual(t, str, "key1=value1&key2=value2")
}

type TestForm struct {
	Ak    *string    `json:"ak"`
	File1 *FileField `json:"file1"`
	File2 *FileField `json:"file2"`
}

func Test_ToFileForm(t *testing.T) {
	file1 := new(FileField).
		SetContent(strings.NewReader("ok")).
		SetContentType("jpg").
		SetFilename("a.jpg")
	body := map[string]interface{}{
		"ak":    "accesskey",
		"file1": file1,
	}
	res := ToFileForm(ToMap(body), "28802961715230")
	byt, err := ioutil.ReadAll(res)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, string(byt), "--28802961715230\r\nContent-Disposition: "+
		"form-data; name=\"ak\"\r\n\r\naccesskey\r\n--28802961715230\r\nContent-Disposition: "+
		"form-data; name=\"file1\"; filename=\"a.jpg\"\r\nContent-Type: jpg\r\n\r\nok\r\n\r\n--28802961715230--\r\n")

	body1 := &TestForm{
		Ak: String("accesskey"),
		File1: &FileField{
			Filename:    String("a.jpg"),
			ContentType: String("jpg"),
			Content:     strings.NewReader("ok"),
		},
	}
	res = ToFileForm(ToMap(body1), "28802961715230")
	byt, err = ioutil.ReadAll(res)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, string(byt), "--28802961715230\r\nContent-Disposition: form-data; "+
		"name=\"ak\"\r\n\r\naccesskey\r\n--28802961715230\r\nContent-Disposition: "+
		"form-data; name=\"file1\"; filename=\"a.jpg\"\r\nContent-Type: jpg\r\n\r\n\r\n\r\n--28802961715230--\r\n")
}

func Test_GetBoundary(t *testing.T) {
	bound := GetBoundary()
	utils.AssertEqual(t, len(bound), 14)
}
