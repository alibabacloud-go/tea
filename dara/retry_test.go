package dara

import (
	// "fmt"
	// "math"
	"math/rand"
	"testing"
)

type AErr struct {
	BaseError
	Code    *string
	Name    *string
	Message *string
}

func (err *AErr) New(obj map[string]interface{}) *AErr {

	err.Name = String("AErr")

	if val, ok := obj["code"].(string); ok {
		err.Code = String(val)
	}

	if val, ok := obj["message"].(string); ok {
		err.Message = String(val)
	}

	return err
}

func (err *AErr) GetCode() *string {
	return err.Code
}

func (err *AErr) GetName() *string {
	return err.Name
}

type BErr struct {
	BaseError
	Code    *string
	Name    *string
	Message *string
}

func (err *BErr) New(obj map[string]interface{}) *BErr {

	err.Name = String("BErr")

	if val, ok := obj["code"].(string); ok {
		err.Code = String(val)
	}

	if val, ok := obj["message"].(string); ok {
		err.Message = String(val)
	}

	return err
}

func (err *BErr) GetCode() *string {
	return err.Code
}

func (err *BErr) GetName() *string {
	return err.Name
}

type CErr struct {
	ResponseError
	Code       *string
	Name       *string
	Message    *string
	RetryAfter *int64
	StatusCode *int
}

func (err *CErr) New(obj map[string]interface{}) *CErr {
	err.Name = String("CErr")

	if val, ok := obj["code"].(string); ok {
		err.Code = String(val)
	}

	if val, ok := obj["message"].(string); ok {
		err.Message = String(val)
	}

	if statusCode, ok := obj["StatusCode"].(int); ok {
		err.StatusCode = Int(statusCode)
	}

	if retryAfter, ok := obj["RetryAfter"].(int64); ok {
		err.RetryAfter = Int64(retryAfter)
	}

	return err
}

func (err *CErr) GetCode() *string {
	return err.Code
}

func (err *CErr) GetName() *string {
	return err.Name
}

func (err *CErr) GetRetryAfter() *int64 {
	return err.RetryAfter
}

func (err *CErr) GetStatusCode() *int {
	return err.StatusCode
}

// BackoffPolicyFactory creates a BackoffPolicy based on the option
func TestBackoffPolicyFactory(t *testing.T) {
	tests := []struct {
		name          string
		option        map[string]interface{}
		expectedError bool
	}{
		{
			name: "Fixed policy",
			option: map[string]interface{}{
				"policy": "Fixed",
			},
			expectedError: false,
		},
		{
			name: "Random policy",
			option: map[string]interface{}{
				"policy": "Random",
			},
			expectedError: false,
		},
		{
			name: "Exponential policy",
			option: map[string]interface{}{
				"policy": "Exponential",
			},
			expectedError: false,
		},
		{
			name: "EqualJitter policy",
			option: map[string]interface{}{
				"policy": "EqualJitter",
			},
			expectedError: false,
		},
		{
			name: "FullJitter policy",
			option: map[string]interface{}{
				"policy": "FullJitter",
			},
			expectedError: false,
		},
		{
			name: "Unknown policy",
			option: map[string]interface{}{
				"policy": "Unknown",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoffPolicy, err := BackoffPolicyFactory(tt.option)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}

			if !tt.expectedError && backoffPolicy == nil {
				t.Errorf("expected a valid BackoffPolicy, got nil")
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		options  RetryOptions
		ctx      RetryPolicyContext
		expected bool
	}{
		{
			name:     "Should not retry when options are nil",
			options:  RetryOptions{},
			ctx:      RetryPolicyContext{},
			expected: true,
		},
		{
			name: "Should not retry when retries exhausted",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"AErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 3,
				Exception:        new(AErr).New(map[string]interface{}{"Code": "A1Err"}),
			},
			expected: false,
		},
		{
			name: "Should retry when conditions match",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"AErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 2,
				Exception:        new(AErr).New(map[string]interface{}{"Code": "A1Err"}),
			},
			expected: true,
		},
		{
			name: "Should retry for different exception",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"AErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 2,
				Exception:        new(BErr).New(map[string]interface{}{"Code": "B1Err"}),
			},
			expected: false,
		},
		{
			name: "Should not retry with no retry condition",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"BErr"}, ErrorCode: []string{"B1Err"}},
				},
				NoRetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"AErr"}, ErrorCode: []string{"B1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 2,
				Exception:        new(AErr).New(map[string]interface{}{"Code": "B1Err"}),
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ShouldRetry(&test.options, &test.ctx)
			if got != test.expected {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

func TestFixedBackoffPolicy(t *testing.T) {

	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "Fixed",
			"period": 1000,
		},
	})

	options := RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1},
	}

	context := RetryPolicyContext{
		RetriesAttempted: 2,
		Exception: new(AErr).New(map[string]interface{}{
			"Code": "A1Err",
		}),
	}

	// Test Delay Time
	expectedDelay := 1000
	if delay := GetBackoffDelay(&options, &context); delay != expectedDelay {
		t.Errorf("Expected delay time %d, got %d", expectedDelay, delay)
	}
}

func TestRandomBackoffPolicy(t *testing.T) {
	// Test case 1: Random backoff policy with period of 1000 and cap of 10000
	rand.Seed(42) // Set seed for reproducibility
	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "Random",
			"period": 1000,
			"cap":    10000,
		},
	})

	options := RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1},
	}

	context := RetryPolicyContext{
		RetriesAttempted: 2,
		Exception: new(AErr).New(map[string]interface{}{
			"Code": "A1Err",
		}),
	}

	// Test Delay Time
	delay := GetBackoffDelay(&options, &context)
	if delay >= 10000 {
		t.Errorf("Expected backoff delay to be less than 10000, got %d", delay)
	}

	// Test case 2: Random backoff policy with period of 10000 and cap of 10
	condition2 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "Random",
			"period": 1000,
			"cap":    10,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition2},
	}

	delay2 := GetBackoffDelay(&options, &context)
	if delay2 != 10 {
		t.Errorf("Expected backoff delay to be 10, got %d", delay2)
	}
}

func TestExponentialBackoffPolicy(t *testing.T) {
	// Test case 1
	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "Exponential",
			"period": 5,
			"cap":    10000,
		},
	})

	options := RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1},
	}

	context := RetryPolicyContext{
		RetriesAttempted: 2,
		Exception: new(AErr).New(map[string]interface{}{
			"Code": "A1Err",
		}),
	}

	// Test Delay Time
	delay := GetBackoffDelay(&options, &context)
	if delay != 1024 {
		t.Errorf("Expected backoff delay to be 1024, got %d", delay)
	}

	// Test case 2
	condition2 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "Exponential",
			"period": 10,
			"cap":    10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition2},
	}

	// Test Delay Time
	delay = GetBackoffDelay(&options, &context)
	if delay != 10000 {
		t.Errorf("Expected backoff delay to be 10000, got %d", delay)
	}
}

func TestEqualJitterBackoff(t *testing.T) {
	rand.Seed(0) // Seed random for predictable results
	// Test case 1
	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "EqualJitter",
			"period": 5,
			"cap":    10000,
		},
	})

	options := RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1},
	}

	context := RetryPolicyContext{
		RetriesAttempted: 2,
		Exception: new(AErr).New(map[string]interface{}{
			"Code": "A1Err",
		}),
	}

	// Test Delay Time
	delay := GetBackoffDelay(&options, &context)
	if delay <= 512 || delay >= 1024 {
		t.Errorf("Expected backoff time in range (512, 1024), got: %d", delay)
	}

	// Test case 2
	condition2 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "ExponentialWithEqualJitter",
			"period": 10,
			"cap":    10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition2},
	}

	// Test Delay Time
	delay = GetBackoffDelay(&options, &context)
	if delay <= 5000 || delay >= 10000 {
		t.Errorf("Expected backoff time in range (5000, 10000), got: %d", delay)
	}
}

func TestFullJitterBackoffPolicy(t *testing.T) {
	// Test case 1
	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "fullJitter",
			"period": 5,
			"cap":    10000,
		},
	})

	options := RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1},
	}

	context := RetryPolicyContext{
		RetriesAttempted: 2,
		Exception: new(AErr).New(map[string]interface{}{
			"Code": "A1Err",
		}),
	}

	// Test Delay Time
	delay := GetBackoffDelay(&options, &context)
	if delay < 0 || delay >= 1024 {
		t.Errorf("Expected backoff time in range [0, 1024), got: %d", delay)
	}

	condition2 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "ExponentialWithFullJitter",
			"period": 10,
			"cap":    10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition2},
	}

	// Test Delay Time
	delay = GetBackoffDelay(&options, &context)
	if delay < 0 || delay >= 10000 {
		t.Errorf("Expected backoff time in range [0, 10000), got: %d", delay)
	}

	// Test case 3 with maxDelay
	condition3 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"MaxDelay":    1000,
		"backoff": map[string]interface{}{
			"policy": "ExponentialWithFullJitter",
			"period": 10,
			"cap":    10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition3},
	}

	// Test Delay Time
	delay = GetBackoffDelay(&options, &context)
	if delay < 0 || delay > 10000 {
		t.Errorf("Expected backoff time in range [0, 10000], got: %d", delay)
	}

	// Test case 4
	condition4 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "ExponentialWithFullJitter",
			"period": 10,
			"cap":    10000 * 10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition4},
	}

	// Test Delay Time
	delay = GetBackoffDelay(&options, &context)
	if delay < 0 || delay > 120*1000 {
		t.Errorf("Expected backoff time in range [0, 120000], got: %d", delay)
	}

}

func TestRetryAfter(t *testing.T) {
	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"CErr"},
		"errorCode":   []string{"CErr"},
		"MaxDelay":    5000,
		"backoff": map[string]interface{}{
			"policy": "EqualJitter",
			"period": 10,
			"cap":    10000,
		},
	})

	options := RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1},
	}

	context := RetryPolicyContext{
		RetriesAttempted: 2,
		Exception: new(CErr).New(map[string]interface{}{
			"Code":       "CErr",
			"RetryAfter": int64(3000),
		}),
	}

	// Test Delay Time
	delay := GetBackoffDelay(&options, &context)
	if delay != 3000 {
		t.Errorf("Expected backoff time must be 3000, got: %d", delay)
	}

	condition2 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"CErr"},
		"errorCode":   []string{"CErr"},
		"maxDelay":    1000,
		"backoff": map[string]interface{}{
			"policy": "EqualJitter",
			"period": 10,
			"cap":    10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition2},
	}

	// Test Delay Time
	delay = GetBackoffDelay(&options, &context)
	if delay != 1000 {
		t.Errorf("Expected backoff time must be 1000, got: %d", delay)
	}
}
