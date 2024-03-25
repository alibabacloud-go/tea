package tea

import (
	"reflect"
	"testing"

	"github.com/alibabacloud-go/tea/v2/utils"
)

func TestBackoffPolicy(t *testing.T) {
	var backoffPolicy BackoffPolicy
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "Any",
	})
	utils.AssertEqual(t, nil, backoffPolicy)
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "Fixed",
		"period": 1000,
	})
	typeOfPolicy := reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "FixedBackoffPolicy", typeOfPolicy.Elem().Name())
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "Random",
		"period": 2,
		"cap":    int64(60 * 1000),
	})
	typeOfPolicy = reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "RandomBackoffPolicy", typeOfPolicy.Elem().Name())
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "Exponential",
		"period": 2,
		"cap":    int64(60 * 1000),
	})
	typeOfPolicy = reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "ExponentialBackoffPolicy", typeOfPolicy.Elem().Name())
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "EqualJitter",
		"period": 2,
		"cap":    int64(60 * 1000),
	})
	typeOfPolicy = reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "EqualJitterBackoffPolicy", typeOfPolicy.Elem().Name())
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "ExponentialWithEqualJitter",
		"period": 2,
		"cap":    int64(60 * 1000),
	})
	typeOfPolicy = reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "EqualJitterBackoffPolicy", typeOfPolicy.Elem().Name())
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "FullJitter",
		"period": 2,
		"cap":    int64(60 * 1000),
	})
	typeOfPolicy = reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "FullJitterBackoffPolicy", typeOfPolicy.Elem().Name())
	backoffPolicy = NewBackoffPolicy(map[string]interface{}{
		"policy": "ExponentialWithFullJitter",
		"period": 2,
		"cap":    int64(60 * 1000),
	})
	typeOfPolicy = reflect.TypeOf(backoffPolicy)
	utils.AssertEqual(t, "FullJitterBackoffPolicy", typeOfPolicy.Elem().Name())
}

func TestFixedBackoffPolicy(t *testing.T) {
	backoffPolicy := FixedBackoffPolicy{
		Period: Int(1000),
	}
	utils.AssertEqual(t, int64(1000), Int64Value(backoffPolicy.GetDelayTimeMillis(nil)))
	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
	}
	utils.AssertEqual(t, int64(1000), Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)))
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
	}
	utils.AssertEqual(t, int64(1000), Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)))
}

func TestRandomBackoffPolicy(t *testing.T) {
	backoffPolicy := RandomBackoffPolicy{
		Period: Int(2),
	}
	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
	}
	utils.AssertEqual(t, true, Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)) < 2)
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
	}
	utils.AssertEqual(t, true, Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)) < 4)
}

func TestExponentialBackoffPolicy(t *testing.T) {
	backoffPolicy := ExponentialBackoffPolicy{
		Period: Int(2),
	}
	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
	}
	utils.AssertEqual(t, int64(4), Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)))
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
	}
	utils.AssertEqual(t, int64(8), Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)))
}

func TestEqualJitterBackoffPolicy(t *testing.T) {
	backoffPolicy := EqualJitterBackoffPolicy{
		Period: Int(2),
	}
	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
	}
	utils.AssertEqual(t, true, Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)) < 5)
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
	}
	utils.AssertEqual(t, true, Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)) < 9)
}

func TestFullJitterBackoffPolicy(t *testing.T) {
	backoffPolicy := FullJitterBackoffPolicy{
		Period: Int(2),
	}
	retryPolicyContext := RetryPolicyContext{
		RetriesAttempted: Int(1),
	}
	utils.AssertEqual(t, true, Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)) < 4)
	retryPolicyContext = RetryPolicyContext{
		RetriesAttempted: Int(2),
	}
	utils.AssertEqual(t, true, Int64Value(backoffPolicy.GetDelayTimeMillis(&retryPolicyContext)) < 8)
}
