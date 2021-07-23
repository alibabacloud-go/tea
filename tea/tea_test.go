package tea

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	Key  string `json:"key,omitempty"`
	Body []byte `json:"body,omitempty"`
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
	Num1      *int          `json:"num1,omitempty" require:"true" minimum:"2"`
	Num2      *int          `json:"num2,omitempty" maximum:"6"`
	Name1     *string       `json:"name1,omitempty" maxLength:"4"`
	Name2     *string       `json:"name2,omitempty" minLength:"2"`
	Str       *string       `json:"str,omitempty" pattern:"[a-d]*" maxLength:"4"`
	MaxLength *errMaxLength `json:"MaxLength,omitempty"`
	MinLength *errMinLength `json:"MinLength,omitempty"`
	Maximum   *errMaximum   `json:"Maximum,omitempty"`
	Minimum   *errMinimum   `json:"Minimum,omitempty"`
	MaxItems  *errMaxItems  `json:"MaxItems,omitempty"`
	MinItems  *errMinItems  `json:"MinItems,omitempty"`
	List      []*string     `json:"list,omitempty" pattern:"[a-d]*" minItems:"2" maxItems:"3" maxLength:"4"`
}

type errMaxLength struct {
	Num *int `json:"num" maxLength:"a"`
}

type errMinLength struct {
	Num *int `json:"num" minLength:"a"`
}

type errMaximum struct {
	Num *int `json:"num" maximum:"a"`
}

type errMinimum struct {
	Num *int `json:"num" minimum:"a"`
}

type errMaxItems struct {
	NumMax []*int `json:"num" maxItems:"a"`
}

type errMinItems struct {
	NumMin []*int `json:"num" minItems:"a"`
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
	err := NewCastError(String("cast error"))
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
		"key":  123,
		"body": []byte("test"),
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "123", out.Key)
	utils.AssertEqual(t, "test", string(out.Body))
}

func TestRuntimeObject(t *testing.T) {
	runtimeobject := NewRuntimeObject(nil)
	utils.AssertNil(t, runtimeobject.IgnoreSSL)

	runtimeobject = NewRuntimeObject(runtimeObj)
	utils.AssertEqual(t, false, BoolValue(runtimeobject.IgnoreSSL))
}

func TestSDKError(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":       "code",
		"statusCode": 404,
		"message":    "message",
		"data": map[string]interface{}{
			"httpCode":  "404",
			"requestId": "dfadfa32cgfdcasd4313",
			"hostId":    "github.com/alibabacloud/tea",
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError:\n   StatusCode: 404\n   Code: code\n   Message: message\n   Data: {\"hostId\":\"github.com/alibabacloud/tea\",\"httpCode\":\"404\",\"requestId\":\"dfadfa32cgfdcasd4313\"}\n", err.Error())

	err.SetErrMsg("test")
	utils.AssertEqual(t, "test", err.Error())
}

func TestSDKErrorCode404(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"statusCode": 404,
		"code":       "NOTFOUND",
		"message":    "message",
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError:\n   StatusCode: 404\n   Code: NOTFOUND\n   Message: message\n   Data: \n", err.Error())
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
	allow := AllowRetry(nil, Int(0))
	utils.AssertEqual(t, true, BoolValue(allow))

	allow = AllowRetry(nil, Int(1))
	utils.AssertEqual(t, false, BoolValue(allow))

	input := map[string]interface{}{
		"retryable":   false,
		"maxAttempts": 2,
	}
	allow = AllowRetry(input, Int(1))
	utils.AssertEqual(t, false, BoolValue(allow))

	input["retryable"] = true
	allow = AllowRetry(input, Int(3))
	utils.AssertEqual(t, false, BoolValue(allow))

	input["retryable"] = true
	allow = AllowRetry(input, Int(1))
	utils.AssertEqual(t, true, BoolValue(allow))
}

func TestMerge(t *testing.T) {
	in := map[string]*string{
		"tea": String("test"),
	}
	valid := map[string]interface{}{
		"valid": "test",
	}
	invalidStr := "sdfdg"
	result := Merge(in, valid, invalidStr)
	utils.AssertEqual(t, "test", StringValue(result["tea"]))
	utils.AssertEqual(t, "test", StringValue(result["valid"]))

	result = Merge(nil)
	utils.AssertEqual(t, map[string]*string{}, result)
}

type Test struct {
	Msg         *string      `json:"Msg,omitempty"`
	Cast        *CastError   `json:"Cast,omitempty"`
	ListPtr     []*string    `json:"ListPtr,omitempty"`
	List        []string     `json:"List,omitempty"`
	CastList    []CastError  `json:"CastList,omitempty"`
	CastListPtr []*CastError `json:"CastListPtr,omitempty"`
	Reader      io.Reader
	Inter       interface{}
}

func TestToMap(t *testing.T) {
	in := map[string]*string{
		"tea": String("test"),
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
			Message: String("message"),
		},
		ListPtr: StringSlice([]string{"test", ""}),
		List:    []string{"list"},
		CastListPtr: []*CastError{
			&CastError{
				Message: String("CastListPtr"),
			},
			nil,
		},
		CastList: []CastError{
			CastError{
				Message: String("CastList"),
			},
		},
		Reader: strings.NewReader(""),
		Inter:  10,
	}
	result = ToMap(valid)
	utils.AssertEqual(t, "tea", result["Msg"])
	utils.AssertNil(t, result["Reader"])
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

	validStr := String(`{"test":"ok"}`)
	result = ToMap(validStr)
	utils.AssertEqual(t, "ok", result["test"])

	validStr1 := String(`{"test":"ok","num":1}`)
	result = ToMap(validStr1)
	utils.AssertEqual(t, "ok", result["test"])

	result = ToMap([]byte(StringValue(validStr)))
	utils.AssertEqual(t, "ok", result["test"])

	result = ToMap([]byte(StringValue(validStr1)))
	utils.AssertEqual(t, "ok", result["test"])

	invalidStr := "sdfdg"
	result = ToMap(invalidStr)
	utils.AssertEqual(t, map[string]interface{}{}, result)

	result = ToMap(10)
	utils.AssertEqual(t, map[string]interface{}{}, result)

	result = ToMap(nil)
	utils.AssertNil(t, result)
}

func Test_Retryable(t *testing.T) {
	ifRetry := Retryable(nil)
	utils.AssertEqual(t, false, BoolValue(ifRetry))

	err := errors.New("tea")
	ifRetry = Retryable(err)
	utils.AssertEqual(t, true, BoolValue(ifRetry))

	errmsg := map[string]interface{}{
		"code": "err",
	}
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, false, BoolValue(ifRetry))

	errmsg["statusCode"] = 400
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, false, BoolValue(ifRetry))

	errmsg["statusCode"] = "400"
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, false, BoolValue(ifRetry))

	errmsg["statusCode"] = 500
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, true, BoolValue(ifRetry))

	errmsg["statusCode"] = "500"
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, true, BoolValue(ifRetry))

	errmsg["statusCode"] = "test"
	err = NewSDKError(errmsg)
	ifRetry = Retryable(err)
	utils.AssertEqual(t, false, BoolValue(ifRetry))
}

func Test_GetBackoffTime(t *testing.T) {
	ms := GetBackoffTime(nil, Int(0))
	utils.AssertEqual(t, 0, IntValue(ms))

	backoff := map[string]interface{}{
		"policy": "no",
	}
	ms = GetBackoffTime(backoff, Int(0))
	utils.AssertEqual(t, 0, IntValue(ms))

	backoff["policy"] = "yes"
	backoff["period"] = 0
	ms = GetBackoffTime(backoff, Int(1))
	utils.AssertEqual(t, 0, IntValue(ms))

	Sleep(Int(1))

	backoff["period"] = 3
	ms = GetBackoffTime(backoff, Int(1))
	utils.AssertEqual(t, true, IntValue(ms) <= 3)
}

var key = `-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBAN5I1VCLYr2IlTLrFpwUGcnwl8yi6V8Mdw+myxfusNgEWiH/FQ4T
AZsIveiLOz9Gcc8m2mZSxst2qGII00scpiECAwEAAQJBAJZEhnA8yjN28eXKJy68
J/LsQrKEL1+h/ZsHFqTHJ6XfiA0CXjbjPsa4jEbpyilMTSgUyoKdJ512ioeco2n6
xUECIQD/JUHaKSuxz55t3efKdppqfopb92mJ2NuPJgrJI70OCwIhAN8HZ0bzr/4a
DLvYCDUKvOj3GzsV1dtBwWuHBaZEafQDAiEAtTnrel//7z5/U55ow4BW0gmrkQM9
bXIhEZ59zryZzl0CIQDFmBqRCu9eshecCP7kd3n88IjopSTOV4iUypBfyXcRnwIg
eXNxUx+BCu2We36+c0deE2+vizL1s6f5XhE6l4bqtiM=
-----END RSA PRIVATE KEY-----`
var cert = `-----BEGIN CERTIFICATE-----
MIIBvDCCAWYCCQDKjNYQxar0mjANBgkqhkiG9w0BAQsFADBlMQswCQYDVQQGEwJh
czEMMAoGA1UECAwDYXNmMQwwCgYDVQQHDANzYWQxCzAJBgNVBAoMAnNkMQ0wCwYD
VQQLDARxd2VyMQswCQYDVQQDDAJzZjERMA8GCSqGSIb3DQEJARYCd2UwHhcNMjAx
MDE5MDI0MDMwWhcNMzAxMDE3MDI0MDMwWjBlMQswCQYDVQQGEwJhczEMMAoGA1UE
CAwDYXNmMQwwCgYDVQQHDANzYWQxCzAJBgNVBAoMAnNkMQ0wCwYDVQQLDARxd2Vy
MQswCQYDVQQDDAJzZjERMA8GCSqGSIb3DQEJARYCd2UwXDANBgkqhkiG9w0BAQEF
AANLADBIAkEA3kjVUItivYiVMusWnBQZyfCXzKLpXwx3D6bLF+6w2ARaIf8VDhMB
mwi96Is7P0ZxzybaZlLGy3aoYgjTSxymIQIDAQABMA0GCSqGSIb3DQEBCwUAA0EA
ZjePopbFugNK0US1MM48V1S2petIsEcxbZBEk/wGqIzrY4RCFKMtbtPSgTDUl3D9
XePemktG22a54ItVJ5FpcQ==
-----END CERTIFICATE-----`
var ca = `-----BEGIN CERTIFICATE-----
MIIBuDCCAWICCQCLw4OWpjlJCDANBgkqhkiG9w0BAQsFADBjMQswCQYDVQQGEwJm
ZDEMMAoGA1UECAwDYXNkMQswCQYDVQQHDAJxcjEKMAgGA1UECgwBZjEMMAoGA1UE
CwwDc2RhMQswCQYDVQQDDAJmZDESMBAGCSqGSIb3DQEJARYDYXNkMB4XDTIwMTAx
OTAyNDQwNFoXDTIzMDgwOTAyNDQwNFowYzELMAkGA1UEBhMCZmQxDDAKBgNVBAgM
A2FzZDELMAkGA1UEBwwCcXIxCjAIBgNVBAoMAWYxDDAKBgNVBAsMA3NkYTELMAkG
A1UEAwwCZmQxEjAQBgkqhkiG9w0BCQEWA2FzZDBcMA0GCSqGSIb3DQEBAQUAA0sA
MEgCQQCxXZTl5IO61Lqd0fBBOSy7ER1gsdA0LkvflP5HEaQygjecLGfrAtD/DWu0
/sxCcBVnQRoP9Yp0ijHJwgXvBnrNAgMBAAEwDQYJKoZIhvcNAQELBQADQQBJF+/4
DEMilhlFY+o9mqCygFVxuvHtQVhpPS938H2h7/P6pXN65jK2Y5hHefZEELq9ulQe
91iBwaQ4e9racCgP
-----END CERTIFICATE-----`

func Test_DoRequest(t *testing.T) {
	origTestHookDo := hookDo
	defer func() { hookDo = origTestHookDo }()
	hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
		return func(req *http.Request) (*http.Response, error) {
			return mockResponse(200, ``, errors.New("Internal error"))
		}
	}
	request := NewRequest()
	request.Method = String("TEA TEST")
	resp, err := DoRequest(request, nil)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `net/http: invalid method "TEA TEST"`, err.Error())

	request.Method = String("")
	request.Protocol = String("https")
	request.Query = map[string]*string{
		"tea": String("test"),
	}
	runtimeObj["httpsProxy"] = "# #%gfdf"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertContains(t, err.Error(), `invalid URL escape "%gf"`)

	request.Pathname = String("?log")
	request.Headers["tea"] = String("")
	request.Headers["content-length"] = nil
	runtimeObj["httpsProxy"] = "http://someuser:somepassword@ecs.aliyun.com"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `Internal error`, err.Error())

	request.Headers["host"] = String("tea-cn-hangzhou.aliyuncs.com:80")
	request.Headers["user-agent"] = String("test")
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertEqual(t, `Internal error`, err.Error())

	runtimeObj["socks5Proxy"] = "# #%gfdf"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, resp)
	utils.AssertContains(t, err.Error(), ` invalid URL escape "%gf"`)

	hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
		return func(req *http.Request) (*http.Response, error) {
			return mockResponse(200, ``, nil)
		}
	}
	runtimeObj["socks5Proxy"] = "socks5://someuser:somepassword@ecs.aliyun.com"
	runtimeObj["localAddr"] = "127.0.0.1"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "test", StringValue(resp.Headers["tea"]))

	runtimeObj["key"] = "private rsa key"
	runtimeObj["cert"] = "private certification"
	runtimeObj["ignoreSSL"] = true
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNotNil(t, err)
	utils.AssertNil(t, resp)

	runtimeObj["key"] = key
	runtimeObj["cert"] = cert
	runtimeObj["ca"] = "private ca"
	runtimeObj["socks5Proxy"] = "socks5://someuser:somepassword@cs.aliyun.com"
	_, err = DoRequest(request, runtimeObj)
	utils.AssertNotNil(t, err)

	runtimeObj["ca"] = ca
	runtimeObj["socks5Proxy"] = "socks5://someuser:somepassword@cs.aliyuncs.com"
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "test", StringValue(resp.Headers["tea"]))

	request.Protocol = String("HTTP")
	runtimeObj["ignoreSSL"] = false
	resp, err = DoRequest(request, runtimeObj)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "test", StringValue(resp.Headers["tea"]))
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
		NoProxy: String("www.aliyun.com"),
	}
	proxy, err := getHttpProxy("http", "www.aliyun.com", runtime)
	utils.AssertNil(t, proxy)
	utils.AssertNil(t, err)

	runtime.NoProxy = nil
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

	runtime.HttpProxy = String("tea2.aliyun.com")
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
	dialcontext := setDialContext(runtime)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	utils.AssertNotNil(t, cancelFunc)
	c, err := dialcontext(ctx, "127.0.0.1", "127.0.0.2")
	utils.AssertNil(t, c)
	utils.AssertEqual(t, "dial 127.0.0.1: unknown network 127.0.0.1", err.Error())

	runtime.LocalAddr = String("127.0.0.1")
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
	reader := ToReader(String(str))
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
	num := 3
	config := &validateTest{
		Num1: &num,
	}
	err := Validate(config)
	utils.AssertNil(t, err)

	err = Validate(new(validateTest))
	utils.AssertEqual(t, err.Error(), "num1 should be setted")

	var tmp *validateTest
	err = Validate(tmp)
	utils.AssertNil(t, err)

	err = Validate(nil)
	utils.AssertNil(t, err)
}

func Test_Recover(t *testing.T) {
	err := Recover(nil)
	utils.AssertNil(t, err)
	defer func() {
		if r := Recover(recover()); r != nil {
			utils.AssertEqual(t, "test", r.Error())
		}
	}()
	panic("test")
}

func Test_validate(t *testing.T) {
	var test *validateTest
	err := validate(reflect.ValueOf(test))
	utils.AssertNil(t, err)

	num := 3
	str0, str1 := "abc", "abcddd"
	val := &validateTest{
		Num1: &num,
		Num2: &num,
		Str:  &str0,
	}

	err = validate(reflect.ValueOf(val))
	utils.AssertNil(t, err)

	val.Str = &str1
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "The length of Str is 6 which is more than 4", err.Error())

	val.Num1 = nil
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "num1 should be setted", err.Error())

	val.Name1 = String("最大长度")
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "num1 should be setted", err.Error())

	val.Num1 = &num
	val.Str = &str0
	val.List = []*string{&str0}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "The length of List is 1 which is less than 2", err.Error())

	val.List = []*string{&str0, &str1}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "The length of List is 6 which is more than 4", err.Error())

	val.List = []*string{&str0, &str0}
	err = validate(reflect.ValueOf(val))
	utils.AssertNil(t, err)

	val.MaxItems = &errMaxItems{
		NumMax: []*int{&num},
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.Atoi: parsing "a": invalid syntax`, err.Error())

	val.MaxItems = nil
	val.MinItems = &errMinItems{
		NumMin: []*int{&num},
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.Atoi: parsing "a": invalid syntax`, err.Error())

	val.MinItems = nil
	val.List = []*string{&str0, &str0, &str0, &str0}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "The length of List is 4 which is more than 3", err.Error())

	str2 := "test"
	val.Str = &str2
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, "test is not matched [a-d]*", err.Error())

	val.Str = &str0
	val.List = []*string{&str0}
	val.MaxLength = &errMaxLength{
		Num: &num,
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.Atoi: parsing "a": invalid syntax`, err.Error())

	val.List = nil
	val.MaxLength = nil
	val.MinLength = &errMinLength{
		Num: &num,
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.Atoi: parsing "a": invalid syntax`, err.Error())

	val.Name2 = String("tea")
	val.MinLength = nil
	val.Maximum = &errMaximum{
		Num: &num,
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.ParseFloat: parsing "a": invalid syntax`, err.Error())

	val.Maximum = nil
	val.Minimum = &errMinimum{
		Num: &num,
	}
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `strconv.ParseFloat: parsing "a": invalid syntax`, err.Error())

	val.Minimum = nil
	val.Num2 = Int(10)
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `The size of Num2 is 10.000000 which is greater than 6.000000`, err.Error())

	val.Num2 = nil
	val.Name1 = String("maxLengthTouch")
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `The length of Name1 is 14 which is more than 4`, err.Error())

	val.Name1 = nil
	val.Name2 = String("")
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `The length of Name2 is 0 which is less than 2`, err.Error())

	val.Name2 = String("tea")
	val.Num1 = Int(0)
	err = validate(reflect.ValueOf(val))
	utils.AssertEqual(t, `The size of Num1 is 0.000000 which is less than 2.000000`, err.Error())
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

func Test_TransInt32AndInt(t *testing.T) {
	a := ToInt(Int32(10))
	utils.AssertEqual(t, IntValue(a), 10)

	b := ToInt32(a)
	utils.AssertEqual(t, Int32Value(b), int32(10))
}
