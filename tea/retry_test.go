package tea

import (
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

type AErrorTest struct {
	BaseError
	Message *string
	Code    *string
}

func (err *AErrorTest) Error() string {
	return "error message"
}

func (err *AErrorTest) ErrorName() *string {
	return String("AErrorTest")
}

func (err *AErrorTest) ErrorCode() *string {
	return String("AErrorTestCode")
}

type BErrorTest struct {
	BaseError
	Message *string
	Code    *string
}

func (err *BErrorTest) Error() string {
	return "error message"
}

func (err *BErrorTest) ErrorName() *string {
	return String("BErrorTest")
}

func (err *BErrorTest) ErrorCode() *string {
	return String("BErrorTestCode")
}

type CErrorTest struct {
	BaseError
	Message *string
	Code    *string
}

func (err *CErrorTest) Error() string {
	return "error message"
}

func (err *CErrorTest) ErrorName() *string {
	return String("CErrorTest")
}

func (err *CErrorTest) ErrorCode() *string {
	return String("CErrorTestCode")
}

func Test_ShouldRetry(t *testing.T) {
	var backoffPolicy BackoffPolicy
	backoffPolicy = &ExponentialBackoffPolicy{
		Period: Int(2),
		Cap:    Int64(60 * 1000),
	}
	retryCondition1 := RetryCondition{
		MaxAttempts: Int(1),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("AErrorTest")},
		ErrorCode:   []*string{String("BErrorTestCode")},
	}

	retryCondition2 := RetryCondition{
		MaxAttempts: Int(2),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("AErrorTest"), String("CErrorTest")},
	}

	retryCondition3 := RetryCondition{
		Exception: []*string{String("BErrorTest")},
	}

	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(0),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(nil, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(nil, &retryPolicyContext)))

	retryOptions := RetryOptions{
		Retryable:        Bool(false),
		RetryCondition:   nil,
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   nil,
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition1, &retryCondition2},
		NoRetryCondition: []*RetryCondition{&retryCondition3},
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(0),
		Error:            &BErrorTest{},
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &BErrorTest{},
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &CErrorTest{},
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
		Error:            &CErrorTest{},
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(3),
		Error:            &CErrorTest{},
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

}

func Test_ThrottlingShouldRetry(t *testing.T) {
	var backoffPolicy BackoffPolicy
	backoffPolicy = &ExponentialBackoffPolicy{
		Period: Int(2),
		Cap:    Int64(60 * 1000),
	}
	retryCondition := RetryCondition{
		MaxAttempts: Int(1),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("ThrottlingError")},
	}

	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            NewError("", map[string]interface{}{}),
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(nil, &retryPolicyContext)))

	retryOptions := RetryOptions{
		Retryable:        Bool(false),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError := NewError("", map[string]interface{}{
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"retryable":  false,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	retryCondition = RetryCondition{
		MaxAttempts: Int(1),
		Backoff:     &backoffPolicy,
		ErrorCode:   []*string{String("Throttling"), String("Throttling.User"), String("Throttling.Api")},
	}
	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, false, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"code":       "Throttling",
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"code":       "Throttling.User",
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"code":       "Throttling.Api",
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))
}

func Test_GetBackoffDelay(t *testing.T) {
	var backoffPolicy BackoffPolicy
	backoffPolicy = &ExponentialBackoffPolicy{
		Period: Int(200),
		Cap:    Int64(60 * 1000),
	}
	retryCondition1 := RetryCondition{
		MaxAttempts: Int(1),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("AErrorTest")},
		ErrorCode:   []*string{String("BErrorTestCode")},
	}

	retryCondition2 := RetryCondition{
		MaxAttempts: Int(2),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("AErrorTest"), String("CErrorTest")},
	}

	retryCondition3 := RetryCondition{
		Exception: []*string{String("BErrorTest")},
	}

	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(0),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(0), Int64Value(GetBackoffDelay(nil, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(nil, &retryPolicyContext)))

	retryOptions := RetryOptions{
		Retryable:        Bool(false),
		RetryCondition:   nil,
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   nil,
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &AErrorTest{},
	}
	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition1, &retryCondition2},
		NoRetryCondition: []*RetryCondition{&retryCondition3},
	}
	utils.AssertEqual(t, int64(400), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(800), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &BErrorTest{},
	}
	utils.AssertEqual(t, int64(400), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            &CErrorTest{},
	}
	utils.AssertEqual(t, int64(400), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
		Error:            &CErrorTest{},
	}
	utils.AssertEqual(t, int64(800), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(3),
		Error:            &CErrorTest{},
	}
	utils.AssertEqual(t, int64(1600), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryCondition4 := RetryCondition{
		MaxAttempts: Int(20),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("AErrorTest")},
	}
	retryOptions = RetryOptions{
		Retryable:      Bool(true),
		RetryCondition: []*RetryCondition{&retryCondition4},
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(10),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(60000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(15),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(60000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	backoffPolicy = &ExponentialBackoffPolicy{
		Period: Int(200),
		Cap:    Int64(180 * 1000),
	}
	retryCondition4 = RetryCondition{
		MaxAttempts: Int(20),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("AErrorTest")},
	}
	retryOptions = RetryOptions{
		Retryable:      Bool(true),
		RetryCondition: []*RetryCondition{&retryCondition4},
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(10),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(120000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(15),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(120000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryCondition4 = RetryCondition{
		MaxAttempts:        Int(20),
		MaxDelayTimeMillis: Int64(30 * 1000),
		Backoff:            &backoffPolicy,
		Exception:          []*string{String("AErrorTest")},
	}
	retryOptions = RetryOptions{
		Retryable:      Bool(true),
		RetryCondition: []*RetryCondition{&retryCondition4},
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(10),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(30000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(15),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(30000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryCondition4 = RetryCondition{
		MaxAttempts: Int(20),
		Backoff:     nil,
		Exception:   []*string{String("AErrorTest")},
	}
	retryOptions = RetryOptions{
		Retryable:      Bool(true),
		RetryCondition: []*RetryCondition{&retryCondition4},
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(10),
		Error:            &AErrorTest{},
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))
}

func Test_GetThrottlingBackoffDelay(t *testing.T) {
	var backoffPolicy BackoffPolicy
	backoffPolicy = &ExponentialBackoffPolicy{
		Period: Int(200),
		Cap:    Int64(60 * 1000),
	}
	retryCondition := RetryCondition{
		MaxAttempts: Int(1),
		Backoff:     &backoffPolicy,
		Exception:   []*string{String("ThrottlingError")},
	}

	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            NewError("", map[string]interface{}{}),
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(nil, &retryPolicyContext)))

	retryOptions := RetryOptions{
		Retryable:        Bool(false),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError := NewError("ThrottlingError", map[string]interface{}{})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(400), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"retryable":  true,
		"retryAfter": int64(320 * 1000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(120000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryCondition = RetryCondition{
		MaxAttempts: Int(1),
		Backoff:     &backoffPolicy,
		ErrorCode:   []*string{String("Throttling"), String("Throttling.User"), String("Throttling.Api")},
	}
	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"code":       "Throttling",
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"code":       "Throttling.User",
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = NewError("ThrottlingError", map[string]interface{}{
		"code":       "Throttling.Api",
		"retryable":  true,
		"retryAfter": int64(2000),
	})
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))
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
