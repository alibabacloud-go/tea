package tea

import (
	"math"
	"math/rand"
	"time"

	"github.com/alibabacloud-go/tea/utils"
)

const (
	// DefaultMaxAttempts sets maximum number of retries
	DefaultMaxAttempts = 3

	// DefaultMinDelay sets minimum retry delay
	DefaultMinDelay = 100 * time.Millisecond

	// DefaultMaxDelayTimeMillis sets maximum retry delay
	DefaultMaxDelay = 120 * time.Second
)

type RetryCondition struct {
	MaxAttempts        *int
	MaxDelayTimeMillis *int64
	Backoff            *BackoffPolicy
	Exception          []*string
	ErrorCode          []*string
}

type RetryOptions struct {
	Retryable        *bool
	RetryCondition   []*RetryCondition
	NoRetryCondition []*RetryCondition
}

type RetryPolicyContext struct {
	RetriesAttempted *int
	Request          *Request
	Response         *Response
	Error            error
}

func ShouldRetry(options *RetryOptions, ctx *RetryPolicyContext) *bool {
	if IntValue(ctx.RetriesAttempted) == 0 {
		return Bool(true)
	}
	if options == nil || !BoolValue(options.Retryable) {
		return Bool(false)
	}

	if err, ok := ctx.Error.(BaseError); ok {
		noRetryConditions := options.NoRetryCondition
		retryConditions := options.RetryCondition
		if noRetryConditions != nil {
			for _, noRetryCondition := range noRetryConditions {
				if utils.Contains(noRetryCondition.Exception, err.ErrorName()) || utils.Contains(noRetryCondition.ErrorCode, err.ErrorCode()) {
					return Bool(false)
				}
			}
		}
		if retryConditions != nil {
			for _, retryCondition := range retryConditions {
				if !utils.Contains(retryCondition.Exception, err.ErrorName()) && !utils.Contains(retryCondition.ErrorCode, err.ErrorCode()) {
					continue
				}
				if IntValue(ctx.RetriesAttempted) > IntValue(retryCondition.MaxAttempts) {
					return Bool(false)
				}
				if err1, ok := err.(*SDKError); ok {
					if BoolValue(err1.Retryable) == false {
						return Bool(false)
					}
				}
				return Bool(true)
			}
		}
	}
	return Bool(false)
}

func GetBackoffDelay(options *RetryOptions, ctx *RetryPolicyContext) *int64 {
	if IntValue(ctx.RetriesAttempted) == 0 {
		return Int64(0)
	}

	if err, ok := ctx.Error.(BaseError); ok {
		if options != nil {
			retryConditions := options.RetryCondition
			if retryConditions != nil {
				for _, retryCondition := range retryConditions {
					if !utils.Contains(retryCondition.Exception, err.ErrorName()) && !utils.Contains(retryCondition.ErrorCode, err.ErrorCode()) {
						continue
					}
					var maxDelay int64
					if retryCondition.MaxDelayTimeMillis != nil {
						maxDelay = Int64Value(retryCondition.MaxDelayTimeMillis)
					} else {
						maxDelay = DefaultMaxDelay.Milliseconds()
					}

					if err1, ok := err.(*SDKError); ok {
						if err1.RetryAfter != nil {
							return Int64(int64(math.Min(float64(Int64Value(err1.RetryAfter)), float64(maxDelay))))
						}
					}

					if retryCondition.Backoff == nil {
						return Int64(DefaultMinDelay.Milliseconds())
					}
					delayTimeMillis := (*retryCondition.Backoff).GetDelayTimeMillis(ctx)
					return Int64(int64(math.Min(float64(Int64Value(delayTimeMillis)), float64(maxDelay))))
				}
			}
		}
	}
	return Int64(DefaultMinDelay.Milliseconds())
}

// Deperacated
func AllowRetry(retry interface{}, retryTimes *int) *bool {
	if IntValue(retryTimes) == 0 {
		return Bool(true)
	}
	retryMap, ok := retry.(map[string]interface{})
	if !ok {
		return Bool(false)
	}
	retryable, ok := retryMap["retryable"].(bool)
	if !ok || !retryable {
		return Bool(false)
	}

	maxAttempts, ok := retryMap["maxAttempts"].(int)
	if !ok || maxAttempts < IntValue(retryTimes) {
		return Bool(false)
	}
	return Bool(true)
}

// Deperacated
func GetBackoffTime(backoff interface{}, retrytimes *int) *int {
	backoffMap, ok := backoff.(map[string]interface{})
	if !ok {
		return Int(0)
	}
	policy, ok := backoffMap["policy"].(string)
	if !ok || policy == "no" {
		return Int(0)
	}

	period, ok := backoffMap["period"].(int)
	if !ok || period == 0 {
		return Int(0)
	}

	maxTime := math.Pow(2.0, float64(IntValue(retrytimes)))
	return Int(rand.Intn(int(maxTime-1)) * period)
}
