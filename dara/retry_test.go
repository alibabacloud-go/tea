package dara

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
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

type retryTimeoutError struct{}

func (retryTimeoutError) Error() string   { return "transport timeout" }
func (retryTimeoutError) Timeout() bool   { return true }
func (retryTimeoutError) Temporary() bool { return true }
func (retryTimeoutError) Is(err error) bool {
	return err == context.DeadlineExceeded
}

func assertTransportRetry(t *testing.T, err error, code string) {
	t.Helper()
	ctx := &RetryPolicyContext{RetriesAttempted: 1, Exception: err}
	for _, field := range []string{"exception", "errorCode"} {
		value := "RetryError"
		if field == "errorCode" {
			value = code
		}
		condition := NewRetryCondition(map[string]interface{}{
			field:         []string{value},
			"maxAttempts": 3,
			"maxDelay":    450,
			"backoff":     map[string]interface{}{"policy": "Fixed", "period": 650},
		})
		options := &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{condition}}
		if !ShouldRetry(options, ctx) {
			t.Errorf("%s: expected retry for %T (%v)", field, err, err)
		}
		if delay := GetBackoffDelay(options, ctx); delay != 450 {
			t.Errorf("%s: expected capped delay 450, got %d", field, delay)
		}
	}
	if ctx.Exception != err || TeaSDKError(err) != err {
		t.Error("the original transport error must be preserved")
	}
}

func TestTransportRetryClassification(t *testing.T) {
	wrap := func(err error) error { return &url.Error{Op: "Post", URL: "http://localhost", Err: err} }
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{"url timeout", wrap(retryTimeoutError{}), "Timeout"},
		{"op timeout", &net.OpError{Op: "read", Net: "tcp", Err: retryTimeoutError{}}, "Timeout"},
		{"wrapped transport timeout", fmt.Errorf("request: %w", wrap(retryTimeoutError{})), "Timeout"},
		{"url EOF", wrap(io.EOF), "EOF"},
		{"url unexpected EOF", wrap(io.ErrUnexpectedEOF), "EOF"},
		{"op EOF", &net.OpError{Op: "read", Net: "tcp", Err: io.EOF}, "EOF"},
	} {
		t.Run(test.name, func(t *testing.T) { assertTransportRetry(t, test.err, test.code) })
	}

	options := &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{{Exception: []string{"RetryError"}}}}
	for _, test := range []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"plain error", errors.New("plain error")},
		{"bare EOF", io.EOF},
		{"bare unexpected EOF", io.ErrUnexpectedEOF},
		{"bare timeout", retryTimeoutError{}},
		{"unsupported scheme", wrap(errors.New("unsupported protocol scheme"))},
		{"certificate error", wrap(x509.UnknownAuthorityError{})},
		{"DNS not found", wrap(&net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{IsNotFound: true}})},
		{"canceled", wrap(context.Canceled)},
		{"wrapped canceled", wrap(fmt.Errorf("request: %w", context.Canceled))},
		{"deadline", wrap(context.DeadlineExceeded)},
		{"op deadline", &net.OpError{Op: "dial", Net: "tcp", Err: context.DeadlineExceeded}},
		{"wrapped deadline", wrap(fmt.Errorf("request: %w", context.DeadlineExceeded))},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := &RetryPolicyContext{RetriesAttempted: 1, Exception: test.err}
			if ShouldRetry(options, ctx) {
				t.Errorf("expected no retry for %T (%v)", test.err, test.err)
			}
		})
	}
}

func TestTransportRetryConditions(t *testing.T) {
	err := &url.Error{Op: "Post", URL: "http://localhost", Err: retryTimeoutError{}}
	condition := &RetryCondition{Exception: []string{"RetryError"}, MaxAttempts: 3}
	for _, test := range []struct {
		name     string
		options  *RetryOptions
		attempts int
		want     bool
	}{
		{"first attempt", nil, 0, true},
		{"nil options", nil, 1, false},
		{"disabled", &RetryOptions{RetryCondition: []*RetryCondition{condition}}, 1, false},
		{"no conditions", &RetryOptions{Retryable: true, MaxAttempts: 5}, 1, false},
		{"HTTP only", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{{Exception: []string{"ServerError"}}}}, 1, false},
		{"matched", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{condition}}, 2, true},
		{"attempt limit", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{condition}}, 3, false},
		{"default limit", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{{Exception: []string{"RetryError"}}}}, 3, false},
		{"code limit", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{{ErrorCode: []string{"Timeout"}, MaxAttempts: 1}}}, 1, false},
		{"excluded name", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{condition}, NoRetryCondition: []*RetryCondition{{Exception: []string{"RetryError"}}}}, 1, false},
		{"excluded code", &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{condition}, NoRetryCondition: []*RetryCondition{{ErrorCode: []string{"Timeout"}}}}, 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := &RetryPolicyContext{RetriesAttempted: test.attempts, Exception: err}
			if got := ShouldRetry(test.options, ctx); got != test.want {
				t.Errorf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestTransportRetryHTTPTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)

	transport := &http.Transport{ResponseHeaderTimeout: 20 * time.Millisecond}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	_, err := client.Get(server.URL)
	var urlErr *url.Error
	if !errors.As(err, &urlErr) || !urlErr.Timeout() {
		t.Fatalf("expected URL timeout, got %T (%v)", err, err)
	}
	assertTransportRetry(t, err, "Timeout")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	deadlineTransport := &http.Transport{}
	defer deadlineTransport.CloseIdleConnections()
	deadlineClient := &http.Client{Transport: deadlineTransport, Timeout: time.Second}
	_, err = deadlineClient.Do(req.WithContext(ctx))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected caller deadline, got %v", err)
	}
	options := &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{{Exception: []string{"RetryError"}}}}
	if ShouldRetry(options, &RetryPolicyContext{RetriesAttempted: 1, Exception: err}) {
		t.Error("caller deadline must not retry")
	}
}

func TestTransportRetryBackoff(t *testing.T) {
	ctx := &RetryPolicyContext{RetriesAttempted: 1, Exception: &url.Error{Err: retryTimeoutError{}}}
	unmatched := &RetryCondition{Exception: []string{"ServerError"}, Backoff: &FixedBackoffPolicy{Period: 900}, MaxDelay: 1000}
	for _, test := range []struct {
		name      string
		condition *RetryCondition
		want      int
	}{
		{"name", &RetryCondition{Exception: []string{"RetryError"}, Backoff: &FixedBackoffPolicy{Period: 250}, MaxDelay: 1000}, 250},
		{"code", &RetryCondition{ErrorCode: []string{"Timeout"}, Backoff: &FixedBackoffPolicy{Period: 350}, MaxDelay: 1000}, 350},
		{"default backoff", &RetryCondition{Exception: []string{"RetryError"}}, MIN_DELAY_TIME},
		{"unmatched", unmatched, MIN_DELAY_TIME},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := &RetryOptions{Retryable: true, RetryCondition: []*RetryCondition{unmatched, test.condition}}
			if got := GetBackoffDelay(options, ctx); got != test.want {
				t.Errorf("expected delay %d, got %d", test.want, got)
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
