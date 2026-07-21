package dara

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/alibabacloud-go/tea/utils"
)

func TestSDKErrorAccessors(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":    "ERR001",
		"message": "test message",
	})
	err.Name = String("CustomError")

	utils.AssertEqual(t, "CustomError", StringValue(err.ErrorName()))
	utils.AssertEqual(t, "test message", StringValue(err.ErrorMessage()))
	utils.AssertEqual(t, "ERR001", StringValue(err.GetCode()))
}

func TestTeaSDKError(t *testing.T) {
	utils.AssertNil(t, TeaSDKError(nil))

	sdkErr := NewSDKError(map[string]interface{}{
		"code":        "code",
		"statusCode":  404,
		"message":     "message",
		"description": "description",
		"detail":      "detail",
		"accessDeniedDetail": map[string]interface{}{
			"AuthAction": "ram:ListUsers",
		},
	})
	converted := TeaSDKError(sdkErr)
	utils.AssertNotNil(t, converted)
	teaErr, ok := converted.(*tea.SDKError)
	utils.AssertEqual(t, true, ok)
	utils.AssertEqual(t, "detail", StringValue(teaErr.Detail))
	utils.AssertEqual(t, true, contains(teaErr.Error(), "Detail: detail"))

	plainErr := NewCastError(String("plain"))
	converted = TeaSDKError(plainErr)
	utils.AssertEqual(t, plainErr, converted)
}

func TestTeaSDKError_ResponseErrorCopiesDetail(t *testing.T) {
	resp := &mockResponseError{
		code:        String("InvalidParameter"),
		statusCode:  Int(400),
		message:     "code: 400, bad request id: r1",
		description: String("desc"),
		detail:      String("field X is invalid"),
		data: map[string]interface{}{
			"Message": "bad",
		},
	}
	converted := TeaSDKError(resp)
	teaErr, ok := converted.(*tea.SDKError)
	utils.AssertEqual(t, true, ok)
	utils.AssertEqual(t, "field X is invalid", StringValue(teaErr.GetDetail()))
	utils.AssertEqual(t, true, contains(teaErr.Error(), "Detail: field X is invalid"))
}

func TestTeaSDKError_ResponseErrorOmitsEmptyDetail(t *testing.T) {
	resp := &mockResponseError{
		code:       String("InvalidParameter"),
		statusCode: Int(400),
		message:    "bad",
		detail:     String("   "),
	}
	converted := TeaSDKError(resp)
	teaErr := converted.(*tea.SDKError)
	utils.AssertNil(t, teaErr.Detail)
	utils.AssertEqual(t, false, contains(teaErr.Error(), "Detail:"))
}

func TestSDKError_ErrorOmitsEmptyDetail(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":    "C",
		"message": "M",
		"detail":  "",
	})
	msg := err.Error()
	utils.AssertEqual(t, false, contains(msg, "Detail:"))
	utils.AssertEqual(t, true, contains(msg, "Code: C"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}

type mockResponseError struct {
	code               *string
	statusCode         *int
	message            string
	description        *string
	detail             *string
	data               map[string]interface{}
	accessDeniedDetail map[string]interface{}
	retryAfter         *int64
	name               *string
}

func (m *mockResponseError) Error() string                              { return m.message }
func (m *mockResponseError) GetName() *string                           { return m.name }
func (m *mockResponseError) GetCode() *string                           { return m.code }
func (m *mockResponseError) GetRetryAfter() *int64                      { return m.retryAfter }
func (m *mockResponseError) GetStatusCode() *int                        { return m.statusCode }
func (m *mockResponseError) GetAccessDeniedDetail() map[string]interface{} {
	return m.accessDeniedDetail
}
func (m *mockResponseError) GetDescription() *string                    { return m.description }
func (m *mockResponseError) GetData() map[string]interface{}            { return m.data }
func (m *mockResponseError) GetDetail() *string                         { return m.detail }
