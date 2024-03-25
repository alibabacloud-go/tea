package tea

import (
	"testing"

	"github.com/alibabacloud-go/tea/v2/utils"
)

type AErrorTest struct {
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

func (err *AErrorTest) RetryAfterTimeMillis() *int64 {
	return nil
}

type BErrorTest struct {
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

func (err *BErrorTest) RetryAfterTimeMillis() *int64 {
	return nil
}

type CErrorTest struct {
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

func (err *CErrorTest) RetryAfterTimeMillis() *int64 {
	return nil
}

type ThrottlingErrorTest struct {
	Code        *string
	StatusCode  *int
	Message     *string
	Description *string
	RetryAfter  *int64
}

func (err *ThrottlingErrorTest) Error() string {
	return StringValue(err.Message)
}

func (err *ThrottlingErrorTest) ErrorName() *string {
	return String("ThrottlingError")
}

func (err *ThrottlingErrorTest) ErrorCode() *string {
	return err.Code
}

func (err *ThrottlingErrorTest) RetryAfterTimeMillis() *int64 {
	return err.RetryAfter
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
		Error:            &ThrottlingErrorTest{},
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
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError := &ThrottlingErrorTest{
		RetryAfter: Int64(int64(2000)),
	}
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

	throttlingError = &ThrottlingErrorTest{
		Code:       String("Throttling"),
		RetryAfter: Int64(int64(2000)),
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = &ThrottlingErrorTest{
		Code:       String("Throttling.User"),
		RetryAfter: Int64(int64(2000)),
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, true, BoolValue(ShouldRetry(&retryOptions, &retryPolicyContext)))

	throttlingError = &ThrottlingErrorTest{
		Code:       String("Throttling.Api"),
		RetryAfter: Int64(int64(2000)),
	}
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
		Error:            &ThrottlingErrorTest{},
	}
	utils.AssertEqual(t, int64(100), Int64Value(GetBackoffDelay(nil, &retryPolicyContext)))

	retryOptions := RetryOptions{
		Retryable:        Bool(false),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(400), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	retryOptions = RetryOptions{
		Retryable:        Bool(true),
		RetryCondition:   []*RetryCondition{&retryCondition},
		NoRetryCondition: nil,
	}
	utils.AssertEqual(t, int64(400), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError := &ThrottlingErrorTest{
		RetryAfter: Int64(int64(2000)),
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = &ThrottlingErrorTest{
		RetryAfter: Int64(int64(320 * 1000)),
	}
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

	throttlingError = &ThrottlingErrorTest{
		Code:       String("Throttling"),
		RetryAfter: Int64(int64(2000)),
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = &ThrottlingErrorTest{
		Code:       String("Throttling.User"),
		RetryAfter: Int64(int64(2000)),
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))

	throttlingError = &ThrottlingErrorTest{
		Code:       String("Throttling.Api"),
		RetryAfter: Int64(int64(2000)),
	}
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(1),
		Error:            throttlingError,
	}
	utils.AssertEqual(t, int64(2000), Int64Value(GetBackoffDelay(&retryOptions, &retryPolicyContext)))
}
