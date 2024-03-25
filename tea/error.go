package tea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// BaseError is an interface for getting actual error
type BaseError interface {
	error
	ErrorName() *string
	ErrorCode() *string
	RetryAfterTimeMillis() *int64
}

// CastError is used for cast type fails
type CastError struct {
	Message *string
	Code    *string
}

// NewCastError is used for cast type fails
func NewCastError(message *string) *CastError {
	return &CastError{
		Message: message,
		Code:    nil,
	}
}

// Return message of CastError
func (err *CastError) Error() string {
	return StringValue(err.Message)
}

func (err *CastError) ErrorName() *string {
	return String("CastError")
}

func (err *CastError) ErrorCode() *string {
	return err.Code
}

func (err *CastError) RetryAfterTimeMillis() *int64 {
	return nil
}

// SDKError struct is used save error code and message
type SDKError struct {
	Code               *string
	StatusCode         *int
	Message            *string
	Data               *string
	Stack              *string
	errMsg             *string
	Description        *string
	AccessDeniedDetail map[string]interface{}
}

// NewSDKError is used for shortly create SDKError object
func NewSDKError(obj map[string]interface{}) *SDKError {
	err := &SDKError{}
	if val, ok := obj["code"].(int); ok {
		err.Code = String(strconv.Itoa(val))
	} else if val, ok := obj["code"].(string); ok {
		err.Code = String(val)
	}

	if obj["message"] != nil {
		err.Message = String(obj["message"].(string))
	}
	if obj["description"] != nil {
		err.Description = String(obj["description"].(string))
	}
	if detail := obj["accessDeniedDetail"]; detail != nil {
		r := reflect.ValueOf(detail)
		if r.Kind().String() == "map" {
			res := make(map[string]interface{})
			tmp := r.MapKeys()
			for _, key := range tmp {
				res[key.String()] = r.MapIndex(key).Interface()
			}
			err.AccessDeniedDetail = res
		}
	}
	if data := obj["data"]; data != nil {
		r := reflect.ValueOf(data)
		if r.Kind().String() == "map" {
			res := make(map[string]interface{})
			tmp := r.MapKeys()
			for _, key := range tmp {
				res[key.String()] = r.MapIndex(key).Interface()
			}
			if statusCode := res["statusCode"]; statusCode != nil {
				if code, ok := statusCode.(int); ok {
					err.StatusCode = Int(code)
				} else if tmp, ok := statusCode.(string); ok {
					code, err_ := strconv.Atoi(tmp)
					if err_ == nil {
						err.StatusCode = Int(code)
					}
				} else if code, ok := statusCode.(*int); ok {
					err.StatusCode = code
				}
			}
		}
		byt := bytes.NewBuffer([]byte{})
		jsonEncoder := json.NewEncoder(byt)
		jsonEncoder.SetEscapeHTML(false)
		jsonEncoder.Encode(data)
		err.Data = String(string(bytes.TrimSpace(byt.Bytes())))
	}

	if statusCode, ok := obj["statusCode"].(int); ok {
		err.StatusCode = Int(statusCode)
	} else if status, ok := obj["statusCode"].(string); ok {
		statusCode, err_ := strconv.Atoi(status)
		if err_ == nil {
			err.StatusCode = Int(statusCode)
		}
	}

	return err
}

// Set ErrMsg by msg
func (err *SDKError) SetErrMsg(msg string) {
	err.errMsg = String(msg)
}

func (err *SDKError) Error() string {
	if err.errMsg == nil {
		str := fmt.Sprintf("SDKError:\n   StatusCode: %d\n   Code: %s\n   Message: %s\n   Data: %s\n",
			IntValue(err.StatusCode), StringValue(err.Code), StringValue(err.Message), StringValue(err.Data))
		err.SetErrMsg(str)
	}
	return StringValue(err.errMsg)
}

func (err *SDKError) ErrorName() *string {
	return String("SDKError")
}

func (err *SDKError) ErrorCode() *string {
	return err.Code
}

func (err *SDKError) RetryAfterTimeMillis() *int64 {
	return nil
}
