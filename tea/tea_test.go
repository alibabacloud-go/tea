package tea

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/utils"
)

type test struct {
	Key  string `json:"key"`
	Body []byte `json:"body"`
}

type PrettifyTest struct {
	name     string
	Strs     []string
	Nums8    []int8
	Unum8    []uint8
	Value    string
	Mapvalue map[string]string
}

var runtimeObj = map[string]interface{}{
	"ignoreSSL":     false,
	"readTimeout":   0,
	"localAddr":     "",
	"httpProxy":     "",
	"httpsProxy":    "",
	"maxIdleConns":  0,
	"socks5Proxy":   "",
	"socks5NetWork": "",
	"listener":      &Progresstest{},
	"tracker":       &utils.ReaderTracker{CompletedBytes: int64(10)},
	"logger":        utils.NewLogger("info", "", &bytes.Buffer{}, "{time}"),
}

type validateTest struct {
	Num  *int       `json:"num" require:"true"`
	Str  *string    `json:"str" pattern:"^[a-d]*$" maxLength:"4"`
	Test *errLength `json:"test"`
	List []*string  `json:"list" pattern:"^[a-d]*$" maxLength:"4"`
}

type errLength struct {
	Num *int `json:"num" maxLength:"a"`
}

type Progresstest struct {
}

func (progress *Progresstest) ProgressChanged(event *utils.ProgressEvent) {
}

func mockResponse(statusCode int, content string, mockerr error) (res *http.Response, err error) {
	status := strconv.Itoa(statusCode)
	res = &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		Header:     map[string][]string{"TEA": []string{"test"}},
		StatusCode: statusCode,
		Status:     status + " " + http.StatusText(statusCode),
	}
	res.Body = ioutil.NopCloser(bytes.NewReader([]byte(content)))
	err = mockerr
	return
}

func TestCastError(t *testing.T) {
	err := NewCastError("cast error")
	utils.AssertEqual(t, "cast error", err.Error())
}

func TestRequest(t *testing.T) {
	request := NewRequest()
	utils.AssertNotNil(t, request)
}

func TestResponse(t *testing.T) {
	httpresponse := &http.Response{
		Body: ioutil.NopCloser(strings.NewReader("response")),
	}
	response := NewResponse(httpresponse)
	utils.AssertNotNil(t, response)

	body, err := response.ReadBody()
	utils.AssertEqual(t, "response", string(body))
	utils.AssertNil(t, err)
}

func TestConvert(t *testing.T) {
	in := map[string]interface{}{
		"key":  "value",
		"body": []byte("test"),
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "value", out.Key)
	utils.AssertEqual(t, "test", string(out.Body))
}

func TestConvertType(t *testing.T) {
	in := map[string]interface{}{
		"key": 123,
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "json: cannot unmarshal number into Go struct field test.key of type string", err.Error())
}

func TestRuntimeObject(t *testing.T) {
	runtimeobject := NewRuntimeObject(nil)
	utils.AssertEqual(t, runtimeobject.IgnoreSSL, false)

	runtimeobject = NewRuntimeObject(runtimeObj)
	utils.AssertEqual(t, false, runtimeobject.IgnoreSSL)
}

func TestSDKError(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":    "code",
		"message": "message",
		"data": map[string]interface{}{
			"httpCode":  "404",
			"requestId": "dfadfa32cgfdcasd4313",
			"hostId":    "github.com/alibabacloud/tea",
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError: code message {\"hostId\":\"github.com/alibabacloud/tea\",\"httpCode\":\"404\",\"requestId\":\"dfadfa32cgfdcasd4313\"}", err.Error())
}

func TestSDKErrorCode404(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":    404,
		"message": "message",
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError: 404 message ", err.Error())
}

func TestToObject(t *testing.T) {
	str := "{sdsfdsd:"
	result := ToObject(str)
	utils.AssertNil(t, result)

	input := map[string]string{
		"name": "test",
	}
	result = ToObject(input)
	utils.AssertEqual(t, "test", result["name"].(string))
}

func TestAllowRetry(t *testing.T) {
	allow := AllowRetry(nil, 0)
	utils.AssertEqual(t, true, allow)

	allow = AllowRetry(nil, 1)
	utils.AssertEqual(t, false, allow)

	input := map[string]interface{}{
		"retryable":   false,
		"maxAttempts": 2,
	}
	allow = AllowRetry(input, 1)
	utils.AssertEqual(t, false, allow)

	input["retryable"] = true
	allow = AllowRetry(input, 3)
	utils.AssertEqual(t, false, allow)

	input["retryable"] = true
	allow = AllowRetry(input, 1)
	utils.AssertEqual(t, true, allow)
}

func TestMerge(t *testing.T) {
	in := map[string]string{
		"tea": "test",
	}
	valid := map[string]interface{}{
		"valid": "test",
	}
	invalidStr := "sdfdg"
	result := Merge(in, valid, invalidStr)
	utils.AssertEqual(t, "test", result["tea"])
	utils.AssertEqual(t, "test", result["valid"])

	result = Merge(nil)
	utils.AssertEqual(t, map[string]string{}, result)
}

type Test struct {
	Msg         *string      `json:"Msg"`
	Cast        *CastError   `json:"Cast"`
	ListPtr     []*string    `json:"ListPtr"`
	List        []string     `json:"List"`
	CastList    []CastError  `json:"CastList"`
	CastListPtr []*CastError `json:"CastListPtr"`
}

func TestToMap(t *testing.T) {
	in := map[string]string{
		"tea": "test",
	}
	result := ToMap(in)
	utils.AssertEqual(t, "test", result["tea"])

	validMap := map[string]interface{}{
		"valid": "test",
	}
	result = ToMap(validMap)
	utils.AssertEqual(t, "test", result["valid"])

	valid := &Test{
		Msg: String("tea"),
		Cast: &CastError{
			Message: "message",
		},
		ListPtr: StringSlice([]string{"test", ""}),
		List:    []string{"list"},
		CastListPtr: []*CastError{
			&CastError{
				Message: "CastListPtr",
			},
			nil,
		},
		CastList: []CastError{
			CastError{
				Message: "CastList",
			},
		},
	}
	result = ToMap(valid)
	utils.AssertEqual(t, "tea", result["Msg"])
	utils.AssertEqual(t, map[string]interface{}{"Message": "message"}, result["Cast"])
	utils.AssertEqual(t, []interface{}{"test", ""}, result["ListPtr"])
	utils.AssertEqual(t, []interface{}{"list"}, result["List"])
	utils.AssertEqual(t, []interface{}{map[string]interface{}{"Message": "CastListPtr"}}, result["CastListPtr"])
	utils.AssertEqual(t, []interface{}{map[string]interface{}{"Message": "CastList"}}, result["CastList"])

	valid1 := &Test{
		Msg: String("tea"),
	}
	result = ToMap(valid1)
	utils.AssertEqual(t, "tea", result["Msg"])

	validStr := `{"test":"ok"}`
	result = ToMap(validStr)
	utils.AssertEqual(t, "ok", result["test"])

	validStr1 := `{"test":"ok","num":1}`
	result = ToMap(validStr1)
	utils.AssertEqual(t, "ok", result["test"])

	result = ToMap([]byte(validStr))
	utils.AssertEqual(t, "ok", result["test"])

	result = ToMap([]byte(validStr1))
	utils.AssertEqual(t, "ok", result["test"])

	invalidStr := "sdfdg"
	result = ToMap(invalidStr)
	utils.AssertEqual(t, result, map[string]interface{}{})

	result = ToMap(10)
	utils.AssertEqual(t, result, map[string]interface{}{})

	result = ToMap(nil)
	utils.AssertEqual(t, map[string]interface{}{}, result)
}

func Test_Retryable(t *testing.T) {
	ifRetry := Retryable(nil)
	utils.AssertEqual(t, false, ifRetry)

	err := errors.New("tea")
	ifRetry = Retryable(err)
	utils.AssertEqual(t, true, ifRetry)

	errmsg := map[string]interface{}{
		"code": "err",
	}
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, true, ifRetry)

	errmsg["code"] = "400"
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, false, ifRetry)
}

func Test_GetBackoffTime(t *testing.T) {
	ms := GetBackoffTime(nil, 0)
	utils.AssertEqual(t, 0, ms)

	backoff := map[string]interface{}{
		"policy": "no",
	}
	ms = GetBackoffTime(backoff, 0)
	utils.AssertEqual(t, 0, ms)

	backoff["policy"] = "yes"
	backoff["period"] = 0
	ms = GetBackoffTime(backoff, 1)
	utils.AssertEqual(t, 0, ms)

	Sleep(1)

	backoff["period"] = 3
	ms = GetBackoffTime(backoff, 1)
	utils.AssertEqual(t, true, ms <= 3)
}

func Test_DoRequest(t *testing.T) {
	origTestHookDo := hookDo
	defer func() { hookDo = origTestHookDo }()
	hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
		return func(req *http.Request) (*http.Response, error) {
			return mockResponse(200, ``, errors.New("Internal error"))
		}
	}
	request := NewRequest()
	request.Port = 80
	request.Method = "TEA TEST"
	resp, err := DoRequest(request, nil)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `net/http: invalid method "TEA TEST"`, err.Error())

	request.Method = ""
	request.Protocol = "https"
	request.Query = map[string]string{
		"tea": "test",
	}
	runtimeObj["httpsProxy"] = "# #%gfdf"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `parse # #%gfdf: invalid URL escape "%gf"`, err.Error())

	request.Pathname = "?log"
	request.Headers["tea"] = ""
	runtimeObj["httpsProxy"] = "http://someuser:somepassword@ecs.aliyun.com"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `Internal error`, err.Error())

	request.Headers["host"] = "tea-cn-hangzhou.aliyuncs.com:80"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `Internal error`, err.Error())

	runtimeObj["socks5Proxy"] = "# #%gfdf"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `parse # #%gfdf: invalid URL escape "%gf"`, err.Error())

	hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
		return func(req *http.Request) (*http.Response, error) {
			return mockResponse(200, ``, nil)
		}
	}
	runtimeObj["socks5Proxy"] = "socks5://someuser:somepassword@ecs.aliyun.com"
	runtimeObj["localAddr"] = "127.0.0.1"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "test", resp.Headers["tea"])
}

func Test_DoRequestWithConcurrent(t *testing.T) {
	origTestHookDo := hookDo
	defer func() { hookDo = origTestHookDo }()
	hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
		return func(req *http.Request) (*http.Response, error) {
			return mockResponse(200, ``, nil)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(readTimeout int) {
			runtime := map[string]interface{}{
				"readTimeout": readTimeout,
			}
			for j := 0; j < 50; j++ {
				wg.Add(1)
				go func() {
					request := NewRequest()
					resp, err := DoRequest(request, runtime)
					utils.AssertNil(t, err)
					utils.AssertNotNil(t, resp)
					wg.Done()
				}()
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func Test_getHttpProxy(t *testing.T) {
	originHttpProxy := os.Getenv("HTTP_PROXY")
	originHttpsProxy := os.Getenv("HTTPS_PROXY")
	originhttpproxy := os.Getenv("http_proxy")
	originhttpsproxy := os.Getenv("https_proxy")
	originNoProxy := os.Getenv("NO_PROXY")
	originnoproxy := os.Getenv("no_proxy")
	defer func() {
		os.Setenv("HTTP_PROXY", originHttpProxy)
		os.Setenv("HTTPS_PROXY", originHttpsProxy)
		os.Setenv("http_proxy", originhttpproxy)
		os.Setenv("https_proxy", originhttpsproxy)
		os.Setenv("NO_PROXY", originNoProxy)
		os.Setenv("no_proxy", originnoproxy)
	}()
	runtime := &RuntimeObject{
		NoProxy: "www.aliyun.com",
	}
	proxy, err := getHttpProxy("http", "www.aliyun.com", runtime)
	utils.AssertNil(t, proxy)
	utils.AssertNil(t, err)

	runtime.NoProxy = ""
	os.Setenv("no_proxy", "tea")
	os.Setenv("http_proxy", "tea.aliyun.com")
	proxy, err = getHttpProxy("http", "www.aliyun.com", runtime)
	utils.AssertEqual(t, "tea.aliyun.com", proxy.Path)
	utils.AssertNil(t, err)

	os.Setenv("NO_PROXY", "tea")
	os.Setenv("HTTP_PROXY", "tea1.aliyun.com")
	proxy, err = getHttpProxy("http", "www.aliyun.com", runtime)
	utils.AssertEqual(t, "tea1.aliyun.com", proxy.Path)
	utils.AssertNil(t, err)

	runtime.HttpProxy = "tea2.aliyun.com"
	proxy, err = getHttpProxy("http", "www.aliyun.com", runtime)
	utils.AssertEqual(t, "tea2.aliyun.com", proxy.Path)
	utils.AssertNil(t, err)

	os.Setenv("no_proxy", "tea")
	os.Setenv("https_proxy", "tea.aliyun.com")
	proxy, err = getHttpProxy("https", "www.aliyun.com", runtime)
	utils.AssertEqual(t, "tea.aliyun.com", proxy.Path)
	utils.AssertNil(t, err)

	os.Setenv("NO_PROXY", "tea")
	os.Setenv("HTTPS_PROXY", "tea1.aliyun.com")
	proxy, err = getHttpProxy("https", "www.aliyun.com", runtime)
	utils.AssertEqual(t, "tea1.aliyun.com", proxy.Path)
	utils.AssertNil(t, err)
}

func Test_SetDialContext(t *testing.T) {
	runtime := &RuntimeObject{}
	dialcontext := setDialContext(runtime, 80)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	utils.AssertNotNil(t, cancelFunc)
	c, err := dialcontext(ctx, "127.0.0.1", "127.0.0.2")
	utils.AssertNil(t, c)
	utils.AssertEqual(t, "dial 127.0.0.1: unknown network 127.0.0.1", err.Error())

	runtime.LocalAddr = "127.0.0.1"
	c, err = dialcontext(ctx, "127.0.0.1", "127.0.0.2")
	utils.AssertNil(t, c)
	utils.AssertEqual(t, "dial 127.0.0.1: unknown network 127.0.0.1", err.Error())
}

func Test_hookdo(t *testing.T) {
	fn := func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("hookdo")
	}
	result := hookDo(fn)
	resp, err := result(nil)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, "hookdo", err.Error())
}

func Test_ToReader(t *testing.T) {
	str := "abc"
	reader := ToReader(str)
	byt, err := ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "abc", string(byt))

	read := strings.NewReader("bcd")
	reader = ToReader(read)
	byt, err = ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "bcd", string(byt))

	byts := []byte("cdf")
	reader = ToReader(byts)
	byt, err = ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "cdf", string(byt))

	num := 10
	defer func() {
		err := recover()
		utils.AssertEqual(t, "Invalid Body. Please set a valid Body.", err.(string))
	}()
	reader = ToReader(num)
	byt, err = ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "", string(byt))
}

func Test_ToString(t *testing.T) {
	str := ToString(10)
	utils.AssertEqual(t, "10", str)

	str = ToString("10")
	utils.AssertEqual(t, "10", str)
}

func Test_Validate(t *testing.T) {
	num := 1
	config := &validateTest{
		Num: &num,
	}
	err := Validate(config)
	utils.AssertNil(t, err)
}

func Test_validate(t *testing.T) {
	var test *validateTest
	err := validate(reflect.ValueOf(test))
	utils.AssertNil(t, err)

	num := 1
	str0, str1 := "acc", "abcddd"
	val := &validateTest{
		Num:  &num,
		Str:  &str0,
		List: []*string{&str0},
	}

	err = validate(reflect.ValueOf(val))
	utils.AssertNil(t, err)

	val.Str = &str1
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "Length of abcddd is more than 4", err.Error())

	val.Num = nil
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "num should be setted", err.Error())

	val.Num = &num
	val.Str = &str0
	val.List = []*string{&str1}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "Length of abcddd is more than 4", err.Error())

	val.Str = nil
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "Length of abcddd is more than 4", err.Error())

	str2 := "test"
	val.Str = &str2
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "test is not matched ^[a-d]*$", err.Error())

	val.Str = &str0
	val.List = []*string{&str0}
	val.Test = &errLength{
		Num: &num,
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.Atoi: parsing "a": invalid syntax`, err.Error())
}

func Test_Prettify(t *testing.T) {
	prettifyTest := &PrettifyTest{
		name:     "prettify",
		Nums8:    []int8{0, 1, 2, 4},
		Unum8:    []uint8{0},
		Value:    "ok",
		Mapvalue: map[string]string{"key": "ccp", "value": "ok"},
	}
	str := Prettify(prettifyTest)
	utils.AssertContains(t, str, "Nums8")

	str = Prettify(nil)
	utils.AssertEqual(t, str, "null")
}
