package dara

import (
	"fmt"
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
			name:    "Should not retry when options pointer semantics: Retryable false",
			options: RetryOptions{Retryable: false},
			ctx: RetryPolicyContext{
				RetriesAttempted: 1,
				Exception:        new(AErr).New(map[string]interface{}{"code": "A1Err"}),
			},
			expected: false,
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
			name: "Should retry when only ErrorCode matches",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"OtherErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 1,
				Exception:        new(AErr).New(map[string]interface{}{"code": "A1Err"}),
			},
			expected: true,
		},
		{
			name: "Should not retry when only ErrorCode matches but attempts exhausted",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 2, Exception: []string{"OtherErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 2,
				Exception:        new(AErr).New(map[string]interface{}{"code": "A1Err"}),
			},
			expected: false,
		},
		{
			name: "Should not retry when no RetryCondition matches",
			options: RetryOptions{
				Retryable:   true,
				MaxAttempts: 5,
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
		{
			name: "Should not retry when NoRetryCondition matches by ErrorCode only",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"AErr"}, ErrorCode: []string{"A1Err"}},
				},
				NoRetryCondition: []*RetryCondition{
					{Exception: []string{"Unrelated"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 1,
				Exception:        new(AErr).New(map[string]interface{}{"code": "A1Err"}),
			},
			expected: false,
		},
		{
			name: "Should not retry when RetryCondition list is empty",
			options: RetryOptions{
				Retryable:      true,
				MaxAttempts:    5,
				RetryCondition: []*RetryCondition{},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 1,
				Exception:        new(BErr).New(map[string]interface{}{"Code": "B1Err"}),
			},
			expected: false,
		},
		{
			name: "Should not retry for non-BaseError even if MaxAttempts remain",
			options: RetryOptions{
				Retryable:   true,
				MaxAttempts: 5,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 3, Exception: []string{"AErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 1,
				Exception:        fmt.Errorf("plain error"),
			},
			expected: false,
		},
		{
			name: "Should stop retrying after default MAX_ATTEMPTS",
			options: RetryOptions{
				Retryable: true,
				RetryCondition: []*RetryCondition{
					{MaxAttempts: 0, Exception: []string{"AErr"}, ErrorCode: []string{"A1Err"}},
				},
			},
			ctx: RetryPolicyContext{
				RetriesAttempted: 3,
				Exception:        new(AErr).New(map[string]interface{}{"Code": "A1Err"}),
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

	t.Run("Should not retry when options is nil pointer", func(t *testing.T) {
		ctx := &RetryPolicyContext{
			RetriesAttempted: 1,
			Exception:        new(AErr).New(map[string]interface{}{"code": "A1Err"}),
		}
		if ShouldRetry(nil, ctx) {
			t.Errorf("expected false when options is nil")
		}
	})
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
	// period * 2^retries: period=5, retries=2 → 5*4=20
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

	delay := GetBackoffDelay(&options, &context)
	if delay != 20 {
		t.Errorf("Expected backoff delay to be 20, got %d", delay)
	}

	// period=1000ms must multiply, not enter the exponent
	condition1b := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "Exponential",
			"period": 1000,
			"cap":    10000,
		},
	})
	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition1b},
	}
	delay = GetBackoffDelay(&options, &context)
	if delay != 4000 {
		t.Errorf("Expected backoff delay to be 4000, got %d", delay)
	}

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

	delay = GetBackoffDelay(&options, &context)
	if delay != 40 {
		t.Errorf("Expected backoff delay to be 40, got %d", delay)
	}
}

func TestEqualJitterBackoff(t *testing.T) {
	rand.Seed(0) // Seed random for predictable results
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

	delay := GetBackoffDelay(&options, &context)
	if delay < 10 || delay > 20 {
		t.Errorf("Expected backoff time in range [10, 20], got: %d", delay)
	}

	condition2 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "ExponentialWithEqualJitter",
			"period": 1000,
			"cap":    10000,
		},
	})

	options = RetryOptions{
		Retryable:      true,
		RetryCondition: []*RetryCondition{condition2},
	}

	delay = GetBackoffDelay(&options, &context)
	if delay < 2000 || delay > 4000 {
		t.Errorf("Expected backoff time in range [2000, 4000], got: %d", delay)
	}
}

func TestFullJitterBackoffPolicy(t *testing.T) {
	condition1 := NewRetryCondition(map[string]interface{}{
		"maxAttempts": 3,
		"exception":   []string{"AErr"},
		"errorCode":   []string{"A1Err"},
		"backoff": map[string]interface{}{
			"policy": "FullJitter",
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

	delay := GetBackoffDelay(&options, &context)
	if delay < 0 || delay >= 20 {
		t.Errorf("Expected backoff time in range [0, 20), got: %d", delay)
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

func TestNewRetryOptions(t *testing.T) {
	options := NewRetryOptions(map[string]interface{}{
		"retryable":   true,
		"maxAttempts": 5,
		"retryCondition": []interface{}{
			map[string]interface{}{
				"maxAttempts": 3,
				"exception":   []string{"AErr"},
				"errorCode":   []string{"A1"},
			},
		},
		"noRetryCondition": []interface{}{
			map[string]interface{}{
				"maxAttempts": 1,
				"exception":   []string{"BErr"},
				"errorCode":   []string{"B1"},
			},
		},
	})
	if options == nil {
		t.Fatal("expected non-nil RetryOptions")
	}
	if !options.Retryable {
		t.Error("expected Retryable to be true")
	}
	if options.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts 5, got %d", options.MaxAttempts)
	}
	if len(options.RetryCondition) != 1 {
		t.Errorf("expected 1 retry condition, got %d", len(options.RetryCondition))
	}
	if len(options.NoRetryCondition) != 1 {
		t.Errorf("expected 1 no-retry condition, got %d", len(options.NoRetryCondition))
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
