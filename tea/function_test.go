package tea

import (
	"errors"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/utils"
)

type PrettifyTest struct {
	name     string
	Strs     []string
	Nums8    []int8
	Unum8    []uint8
	Value    string
	Mapvalue map[string]string
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

func Test_SleepMillis(t *testing.T) {
	start := time.Now()
	SleepMillis(Int64(1000))
	SleepMillis(Int64(0))
	SleepMillis(nil)
	cost := time.Since(start)
	utils.AssertEqual(t, cost.Seconds() >= 1, true)
}

func Test_Merge(t *testing.T) {
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

func Test_Convert(t *testing.T) {
	in := map[string]interface{}{
		"key":  "value",
		"body": []byte("test"),
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "value", out.Key)
	utils.AssertEqual(t, "test", string(out.Body))

	in = map[string]interface{}{
		"key":  123,
		"body": []byte("test"),
	}
	out = new(test)
	err = Convert(in, &out)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "123", out.Key)
	utils.AssertEqual(t, "test", string(out.Body))
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
