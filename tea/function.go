package tea

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func Prettify(i interface{}) string {
	resp, _ := json.MarshalIndent(i, "", "   ")
	return string(resp)
}

func Sleep(backoffTime *int64) {
	sleeptime := time.Duration(Int64Value(backoffTime)) * time.Millisecond
	time.Sleep(sleeptime)
}

func Merge(args ...interface{}) map[string]*string {
	finalArg := make(map[string]*string)
	for _, obj := range args {
		switch obj.(type) {
		case map[string]*string:
			arg := obj.(map[string]*string)
			for key, value := range arg {
				if value != nil {
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
					finalArg[key] = String(value)
				}
			}
		}
	}

	return finalArg
}

// Convert is use convert map[string]interface object to struct
func Convert(in interface{}, out interface{}) error {
	byt, _ := json.Marshal(in)
	decoder := jsonParser.NewDecoder(bytes.NewReader(byt))
	decoder.UseNumber()
	err := decoder.Decode(&out)
	return err
}

// Recover is used to format error
func Recover(in interface{}) error {
	if in == nil {
		return nil
	}
	return errors.New(fmt.Sprint(in))
}

// Deprecated
func Retryable(err error) *bool {
	if err == nil {
		return Bool(false)
	}
	if realErr, ok := err.(*SDKError); ok {
		if realErr.StatusCode == nil {
			return Bool(false)
		}
		code := IntValue(realErr.StatusCode)
		return Bool(code >= http.StatusInternalServerError)
	}
	return Bool(true)
}
