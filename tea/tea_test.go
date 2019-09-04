package tea

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/utils"
)

type test struct {
	Key string `json:"key"`
}

func mockResponse(statusCode int, content string, mockerr error) (res *http.Response, err error) {
	status := strconv.Itoa(statusCode)
	res = &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		Header:     map[string][]string{"tea": []string{"test"}},
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
		"key": "value",
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "value", out.Key)
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
	obj := map[string]interface{}{
		"ignoreSSL": 10,
	}
	runtimeobject := NewRuntimeObject(obj)
	utils.AssertNil(t, runtimeobject)

	obj = map[string]interface{}{
		"ignoreSSL": false,
	}
	runtimeobject = NewRuntimeObject(obj)
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
	utils.AssertEqual(t, "SDKError: {\"hostId\":\"github.com/alibabacloud/tea\",\"httpCode\":\"404\",\"requestId\":\"dfadfa32cgfdcasd4313\"} message ", err.Error())
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
		"retryable":    false,
		"max-attempts": 2,
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

func TestToMap(t *testing.T) {
	in := map[string]string{
		"tea": "test",
	}
	validMap := map[string]interface{}{
		"valid": "test",
	}
	valid := &CastError{
		Message: "tea",
	}

	invalidStr := "sdfdg"
	result := ToMap(in, validMap, valid, invalidStr)
	utils.AssertEqual(t, "test", result["tea"])
	utils.AssertEqual(t, "test", result["valid"])

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
	times := GetBackoffTime(nil)
	utils.AssertEqual(t, 0, times)

	backoff := map[string]interface{}{
		"policy": "no",
	}
	times = GetBackoffTime(backoff)
	utils.AssertEqual(t, 0, times)

	backoff["policy"] = "yes"
	backoff["period"] = 0
	times = GetBackoffTime(backoff)
	utils.AssertEqual(t, 0, times)
	Sleep(1)

	backoff["period"] = 3
	times = GetBackoffTime(backoff)
	utils.AssertEqual(t, 3, times)
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
	runtime := map[string]interface{}{
		"httpsProxy": "# #%gfdf",
	}
	resp, err = DoRequest(request, runtime)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `parse # #%gfdf: invalid URL escape "%gf"`, err.Error())

	request.Pathname = "?log"
	request.Headers["tea"] = ""
	runtime["httpsProxy"] = "http://someuser:somepassword@ecs.aliyun.com"
	resp, err = DoRequest(request, runtime)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `Internal error`, err.Error())

	runtime["socks5Proxy"] = "# #%gfdf"
	resp, err = DoRequest(request, runtime)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `parse # #%gfdf: invalid URL escape "%gf"`, err.Error())

	hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
		return func(req *http.Request) (*http.Response, error) {
			return mockResponse(200, ``, nil)
		}
	}
	runtime["socks5Proxy"] = "socks5://someuser:somepassword@ecs.aliyun.com"
	runtime["localAddr"] = "127.0.0.1"
	resp, err = DoRequest(request, runtime)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "test", resp.Headers["tea"])
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
	dialcontext := SetDialContext(runtime, 80)
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
