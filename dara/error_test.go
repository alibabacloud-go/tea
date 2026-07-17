package dara

import (
	"testing"

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
		"code":       "code",
		"statusCode": 404,
		"message":    "message",
		"description": "description",
		"detail":      "detail",
		"accessDeniedDetail": map[string]interface{}{
			"AuthAction": "ram:ListUsers",
		},
	})
	converted := TeaSDKError(sdkErr)
	utils.AssertNotNil(t, converted)

	plainErr := NewCastError(String("plain"))
	converted = TeaSDKError(plainErr)
	utils.AssertEqual(t, plainErr, converted)
}
