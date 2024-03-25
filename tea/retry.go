package tea

import (
	"math"
	"time"

	"github.com/alibabacloud-go/tea/v2/utils"
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
					if err.RetryAfterTimeMillis() != nil {
						return Int64(int64(math.Min(float64(Int64Value(err.RetryAfterTimeMillis())), float64(maxDelay))))
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
