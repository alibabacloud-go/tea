package tea

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/alibabacloud-go/debug/debug"
)

var debugLog = debug.Init("tea")

// CastError is used for cast type fails
type CastError struct {
	Message string
}

// NewCastError is used for cast type fails
func NewCastError(message string) (err error) {
	return &CastError{
		Message: message,
	}
}

func (err *CastError) Error() string {
	return err.Message
}

func firstDownCase(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}

// Convert is use convert map[string]interface object to struct
func Convert(in map[string]interface{}, out interface{}) error {
	v := reflect.ValueOf(out).Elem()
	if v.Kind() != reflect.Ptr {
		return NewCastError("The out parameter must be pointer")
	}
	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	for i := 0; i < v.Elem().NumField(); i++ {
		fieldInfo := v.Elem().Type().Field(i)
		name, _ := fieldInfo.Tag.Lookup("json")
		name = upLetter(name)
		if value, ok := in[name]; ok {
			if reflect.ValueOf(value).Kind() == v.Elem().FieldByName(fieldInfo.Name).Kind() {
				v.Elem().FieldByName(fieldInfo.Name).Set(reflect.ValueOf(value))
			} else {
				currentType := reflect.ValueOf(value).Type()
				expectType := v.Elem().FieldByName(fieldInfo.Name).Type()
				return NewCastError(fmt.Sprintf("Convert type fails for field: %s, expect type: %s, current type: %s", name, expectType, currentType))
			}
		}
	}

	out = v.Interface()
	return nil
}

func upLetter(name string) string {
	strs := strings.Split(name, "-")
	for key, value := range strs {
		if len(strs) >= 2 {
			strs[key] = strings.ToUpper(string(value[0])) + value[1:]
		}
	}
	name = strings.Join(strs, "-")
	return name
}

// Request is used wrap http request
type Request struct {
	Protocol string
	Port     int
	Method   string
	Pathname string
	Headers  map[string]string
	Query    map[string]string
	Body     string
}

// NewRequest is used shortly create Request
func NewRequest() (req *Request) {
	return &Request{
		Headers: map[string]string{},
		Query:   map[string]string{},
	}
}

// Response is use d wrap http response
type Response struct {
	*http.Response
	StatusCode    int
	StatusMessage string
}

// NewResponse is create response with http response
func NewResponse(httpResponse *http.Response) (res *Response) {
	res = &Response{
		Response: httpResponse,
	}
	res.StatusCode = httpResponse.StatusCode
	res.StatusMessage = httpResponse.Status
	return
}

// ReadBody is used read response body
func (response *Response) ReadBody() (body []byte, err error) {
	defer response.Body.Close()
	body, err = ioutil.ReadAll(response.Body)
	return
}

// DoRequest is used send request to server
func DoRequest(request *Request) (response *Response, err error) {
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
		requestURL = fmt.Sprintf("%s?%s", requestURL, querystring)
	}

	debugLog(requestMethod)
	debugLog(requestURL)
	httpRequest, err := http.NewRequest(requestMethod, requestURL, strings.NewReader(request.Body))
	if err != nil {
		return
	}

	for key, value := range request.Headers {
		httpRequest.Header[key] = []string{value}
		debugLog("> %s: %s", key, value)
	}
	httpRequest.Host = domain

	httpClient := &http.Client{}
	res, err := httpClient.Do(httpRequest)
	if res != nil {
		debugLog("< HTTP/1.1 %s", res.Status)
		for key, value := range res.Header {
			debugLog("< %s: %s", key, strings.Join(value, ""))
		}
	}

	if err != nil {
		return
	}

	response = NewResponse(res)
	return
}

// SDKError struct is used save error code and message
type SDKError struct {
	Code    string
	Message string
	Data    string
}

func (err *SDKError) Error() string {
	return fmt.Sprintf("SDKError: %s %s %s", err.Code, err.Message, err.Data)
}

func AllowRetry(retry interface{}, retryTimes int) bool {
	retryMap, ok := retry.(map[string]interface{})
	if !ok {
		return false
	}
	retryable, ok := retryMap["retryable"].(bool)
	if !ok || retryable {
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
			json.Unmarshal(byt, &arg)
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
		code, _ := strconv.Atoi(realErr.Code)
		return code >= http.StatusInternalServerError
	}
	return true
}

func GetBackoffTime(backoff interface{}, retryTimes int) int {
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
