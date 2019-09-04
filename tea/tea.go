package tea

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alibabacloud-go/debug/debug"
	"golang.org/x/net/proxy"
)

var debugLog = debug.Init("tea")

var hookDo = func(fn func(req *http.Request) (*http.Response, error)) func(req *http.Request) (*http.Response, error) {
	return fn
}

// CastError is used for cast type fails
type CastError struct {
	Message string
}

// Request is used wrap http request
type Request struct {
	Protocol string
	Port     int
	Method   string
	Pathname string
	Headers  map[string]string
	Query    map[string]string
	Body     io.Reader
}

// Response is use d wrap http response
type Response struct {
	*http.Response
	StatusCode    int
	StatusMessage string
	Headers       map[string]string
}

// SDKError struct is used save error code and message
type SDKError struct {
	Code    string
	Message string
	Data    string
}

// RuntimeObject is used for converting http configuration
type RuntimeObject struct {
	IgnoreSSL      bool   `json:"ignoreSSL" xml:"ignoreSSL"`
	ReadTimeout    int    `json:"readTimeout" xml:"readTimeout"`
	ConnectTimeout int    `json:"connectTimeout" xml:"connectTimeout"`
	LocalAddr      string `json:"localAddr" xml:"localAddr"`
	HttpProxy      string `json:"httpProxy" xml:"httpProxy"`
	HttpsProxy     string `json:"httpsProxy" xml:"httpsProxy"`
	NoProxy        string `json:"noProxy" xml:"noProxy"`
	MaxIdleConns   int    `json:"maxIdleConns" xml:"maxIdleConns"`
	Socks5Proxy    string `json:"socks5Proxy" xml:"socks5Proxy"`
	Socks5NetWork  string `json:"socks5NetWork" xml:"socks5NetWork"`
}

// NewCastError is used for cast type fails
func NewCastError(message string) (err error) {
	return &CastError{
		Message: message,
	}
}

// NewRequest is used shortly create Request
func NewRequest() (req *Request) {
	return &Request{
		Headers: map[string]string{},
		Query:   map[string]string{},
	}
}

// NewResponse is create response with http response
func NewResponse(httpResponse *http.Response) (res *Response) {
	res = &Response{
		Response: httpResponse,
	}
	res.Headers = make(map[string]string)
	res.StatusCode = httpResponse.StatusCode
	res.StatusMessage = httpResponse.Status
	return
}

// NewSDKError is used for shortly create SDKError object
func NewSDKError(obj map[string]interface{}) *SDKError {
	err := &SDKError{}
	if val, ok := obj["code"].(int); ok {
		err.Code = strconv.Itoa(val)
	} else if val, ok := obj["code"].(string); ok {
		err.Code = val
	}

	if obj["message"] != nil {
		err.Message = obj["message"].(string)
	}
	if data := obj["data"]; data != nil {
		byt, _ := json.Marshal(data)
		err.Code = string(byt)
	}
	return err
}

// NewRuntimeObject is used for shortly create runtime object
func NewRuntimeObject(obj map[string]interface{}) *RuntimeObject {
	runtimeObject := new(RuntimeObject)
	byt, _ := json.Marshal(obj)
	err := json.Unmarshal(byt, runtimeObject)
	if err != nil {
		return nil
	}
	return runtimeObject
}

// Return message of CastError
func (err *CastError) Error() string {
	return err.Message
}

// Convert is use convert map[string]interface object to struct
func Convert(in interface{}, out interface{}) error {
	byt, _ := json.Marshal(in)
	err := json.Unmarshal(byt, out)
	return err
}

// ReadBody is used read response body
func (response *Response) ReadBody() (body []byte, err error) {
	defer response.Body.Close()
	body, err = ioutil.ReadAll(response.Body)
	return
}

// DoRequest is used send request to server
func DoRequest(request *Request, requestRuntime map[string]interface{}) (response *Response, err error) {
	runtimeObject := NewRuntimeObject(requestRuntime)
	requestMethod := request.Method
	if requestMethod == "" {
		requestMethod = "GET"
	}

	protocol := "http"
	if request.Protocol != "" {
		protocol = strings.ToLower(request.Protocol)
	}

	port := 0
	if protocol == "http" {
		port = 80
	} else if protocol == "https" {
		port = 443
	}

	if request.Port != 0 {
		port = request.Port
	}

	domain := request.Headers["host"]
	requestURL := fmt.Sprintf("%s://%s:%d%s", protocol, domain, port, request.Pathname)
	queryParams := request.Query
	// sort QueryParams by key
	q := url.Values{}
	for key, value := range queryParams {
		q.Add(key, value)
	}
	querystring := q.Encode()
	if len(querystring) > 0 {
		if strings.Contains(requestURL, "?") {
			requestURL = fmt.Sprintf("%s&%s", requestURL, querystring)
		} else {
			requestURL = fmt.Sprintf("%s?%s", requestURL, querystring)
		}
	}
	debugLog(requestMethod)
	debugLog(requestURL)

	httpRequest, err := http.NewRequest(requestMethod, requestURL, request.Body)
	if err != nil {
		return
	}
	httpRequest.Host = domain

	httpClient := &http.Client{}
	trans := new(http.Transport)
	httpClient.Timeout = time.Duration(runtimeObject.ConnectTimeout) * time.Second
	httpProxy, err := getHttpProxy(protocol, domain, runtimeObject)
	if err != nil {
		return
	}
	trans.MaxIdleConns = runtimeObject.MaxIdleConns
	trans.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: runtimeObject.IgnoreSSL,
	}
	if httpProxy != nil {
		trans.Proxy = http.ProxyURL(httpProxy)
		if httpProxy.User != nil {
			password, _ := httpProxy.User.Password()
			auth := httpProxy.User.Username() + ":" + password
			basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			request.Headers["Proxy-Authorization"] = basic
		}
	}
	for key, value := range request.Headers {
		if value == "" {
			continue
		}
		httpRequest.Header[key] = []string{value}
		debugLog("> %s: %s", key, value)
	}
	if runtimeObject.Socks5Proxy != "" {
		socks5Proxy, err := getSocks5Proxy(runtimeObject)
		if err != nil {
			return nil, err
		}
		if socks5Proxy != nil {
			var auth *proxy.Auth
			if socks5Proxy.User != nil {
				password, _ := socks5Proxy.User.Password()
				auth = &proxy.Auth{
					User:     socks5Proxy.User.Username(),
					Password: password,
				}
			}
			dialer, err := proxy.SOCKS5(strings.ToLower(runtimeObject.Socks5NetWork), socks5Proxy.String(), auth,
				&net.Dialer{
					Timeout:   time.Duration(runtimeObject.ConnectTimeout) * time.Second,
					DualStack: true,
					LocalAddr: getLocalAddr(runtimeObject.LocalAddr, port),
				})
			if err != nil {
				return nil, err
			}
			trans.Dial = dialer.Dial
		}
	} else {
		trans.DialContext = SetDialContext(runtimeObject, port)
	}
	httpClient.Transport = trans

	res, err := hookDo(httpClient.Do)(httpRequest)
	if err != nil {
		return
	}

	response = NewResponse(res)
	debugLog("< HTTP/1.1 %s", res.Status)
	for key, value := range res.Header {
		debugLog("< %s: %s", key, strings.Join(value, ""))
		if len(value) != 0 {
			response.Headers[key] = value[0]
		}
	}
	return
}

func getNoProxy(protocol string, runtime *RuntimeObject) []string {
	var urls []string
	if runtime.NoProxy != "" {
		urls = strings.Split(runtime.NoProxy, ",")
	} else if rawurl := os.Getenv("NO_PROXY"); rawurl != "" {
		urls = strings.Split(rawurl, ",")
	} else if rawurl := os.Getenv("no_proxy"); rawurl != "" {
		urls = strings.Split(rawurl, ",")
	}

	return urls
}

func getHttpProxy(protocol, host string, runtime *RuntimeObject) (proxy *url.URL, err error) {
	urls := getNoProxy(protocol, runtime)
	for _, url := range urls {
		if url == host {
			return nil, nil
		}
	}
	if protocol == "https" {
		if runtime.HttpsProxy != "" {
			proxy, err = url.Parse(runtime.HttpsProxy)
		} else if rawurl := os.Getenv("HTTPS_PROXY"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		} else if rawurl := os.Getenv("https_proxy"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		}
	} else {
		if runtime.HttpProxy != "" {
			proxy, err = url.Parse(runtime.HttpProxy)
		} else if rawurl := os.Getenv("HTTP_PROXY"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		} else if rawurl := os.Getenv("http_proxy"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		}
	}

	return proxy, err
}

func getSocks5Proxy(runtime *RuntimeObject) (proxy *url.URL, err error) {
	if runtime.Socks5Proxy != "" {
		proxy, err = url.Parse(runtime.Socks5Proxy)
	}
	return proxy, err
}

func getLocalAddr(localAddr string, port int) (addr *net.TCPAddr) {
	if localAddr != "" {
		addr = &net.TCPAddr{
			Port: port,
			IP:   []byte(localAddr),
		}
	}
	return addr
}

func SetDialContext(runtime *RuntimeObject, port int) func(cxt context.Context, net, addr string) (c net.Conn, err error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if runtime.LocalAddr != "" {
			netAddr := &net.TCPAddr{
				Port: port,
				IP:   []byte(runtime.LocalAddr),
			}
			return (&net.Dialer{
				Timeout:   time.Duration(runtime.ConnectTimeout) * time.Second,
				DualStack: true,
				LocalAddr: netAddr,
			}).DialContext(ctx, network, address)
		}
		return (&net.Dialer{
			Timeout:   time.Duration(runtime.ConnectTimeout) * time.Second,
			DualStack: true,
		}).DialContext(ctx, network, address)
	}
}

func (err *SDKError) Error() string {
	return fmt.Sprintf("SDKError: %s %s %s", err.Code, err.Message, err.Data)
}

func ToObject(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	byt, err := json.Marshal(obj)
	if err != nil {
		return nil
	}
	err = json.Unmarshal(byt, &result)
	if err != nil {
		return nil
	}
	return result
}

func AllowRetry(retry interface{}, retryTimes int) bool {
	if retryTimes == 0 {
		return true
	}
	retryMap, ok := retry.(map[string]interface{})
	if !ok {
		return false
	}
	retryable, ok := retryMap["retryable"].(bool)
	if !ok || !retryable {
		return false
	}

	max_attempts, ok := retryMap["max-attempts"].(int)
	if !ok || max_attempts < retryTimes {
		return false
	}
	return true
}

func Merge(args ...interface{}) map[string]string {
	finalArg := make(map[string]string)
	for _, obj := range args {
		switch obj.(type) {
		case map[string]string:
			arg := obj.(map[string]string)
			for key, value := range arg {
				if value != "" {
					finalArg[key] = value
				}
			}
		default:
			byt, _ := json.Marshal(obj)
			arg := make(map[string]string)
			err := json.Unmarshal(byt, &arg)
			if err != nil {
				return finalArg
			}
			for key, value := range arg {
				if value != "" {
					finalArg[key] = value
				}
			}
		}
	}

	return finalArg
}

func ToMap(args ...interface{}) map[string]interface{} {
	finalArg := make(map[string]interface{})
	for _, obj := range args {
		switch obj.(type) {
		case map[string]string:
			arg := obj.(map[string]string)
			for key, value := range arg {
				if value != "" {
					finalArg[key] = value
				}
			}
		case map[string]interface{}:
			arg := obj.(map[string]interface{})
			for key, value := range arg {
				if value != "" {
					finalArg[key] = value
				}
			}
		default:
			byt, _ := json.Marshal(obj)
			arg := make(map[string]interface{})
			err := json.Unmarshal(byt, &arg)
			if err != nil {
				return finalArg
			}
			for key, value := range arg {
				if value != "" {
					finalArg[key] = value
				}
			}
		}
	}

	return finalArg
}

func Retryable(err error) bool {
	if err == nil {
		return false
	}
	if realErr, ok := err.(*SDKError); ok {
		code, err := strconv.Atoi(realErr.Code)
		if err != nil {
			return true
		}
		return code >= http.StatusInternalServerError
	}
	return true
}

func GetBackoffTime(backoff interface{}) int {
	backoffMap, ok := backoff.(map[string]interface{})
	if !ok {
		return 0
	}
	policy, ok := backoffMap["policy"].(string)
	if !ok || policy == "no" {
		return 0
	}

	period, ok := backoffMap["period"].(int)
	if !ok || period == 0 {
		return 0
	}
	return period
}

func Sleep(backoffTime int) {
	sleeptime := time.Duration(backoffTime) * time.Second
	time.Sleep(sleeptime)
}
