package tea

import (
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

func TestCastError(t *testing.T) {
	var err BaseError
	err = NewCastError(String("cast error"))
	utils.AssertEqual(t, "cast error", err.Error())
	utils.AssertEqual(t, "", StringValue(err.ErrorCode()))
	utils.AssertEqual(t, "CastError", StringValue(err.ErrorName()))
}

func TestSDKError(t *testing.T) {
	var err0 BaseError
	err0 = NewSDKError(map[string]interface{}{
		"code":       "code",
		"statusCode": 404,
		"message":    "message",
		"data": map[string]interface{}{
			"httpCode":  "404",
			"requestId": "dfadfa32cgfdcasd4313",
			"hostId":    "github.com/alibabacloud/tea",
			"recommend": "https://中文?q=a.b&product=c&requestId=123",
		},
		"description": "description",
		"accessDeniedDetail": map[string]interface{}{
			"AuthAction":        "ram:ListUsers",
			"AuthPrincipalType": "SubUser",
			"PolicyType":        "ResourceGroupLevelIdentityBassdPolicy",
			"NoPermissionType":  "ImplicitDeny",
			"UserId":            123,
		},
	})
	utils.AssertNotNil(t, err0)
	utils.AssertEqual(t, "SDKError:\n   StatusCode: 404\n   Code: code\n   Message: message\n   Data: {\"hostId\":\"github.com/alibabacloud/tea\",\"httpCode\":\"404\",\"recommend\":\"https://中文?q=a.b&product=c&requestId=123\",\"requestId\":\"dfadfa32cgfdcasd4313\"}\n", err0.Error())
	var err *SDKError
	err = err0.(*SDKError)
	err.SetErrMsg("test")
	utils.AssertEqual(t, "test", err.Error())
	utils.AssertEqual(t, 404, *err.StatusCode)
	utils.AssertEqual(t, "description", *err.Description)
	utils.AssertEqual(t, "ImplicitDeny", err.AccessDeniedDetail["NoPermissionType"])
	utils.AssertEqual(t, 123, err.AccessDeniedDetail["UserId"])

	err = NewSDKError(map[string]interface{}{
		"statusCode": "404",
		"data": map[string]interface{}{
			"statusCode": 500,
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, 404, *err.StatusCode)

	err = NewSDKError(map[string]interface{}{
		"data": map[string]interface{}{
			"statusCode": 500,
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, 500, *err.StatusCode)

	err = NewSDKError(map[string]interface{}{
		"data": map[string]interface{}{
			"statusCode": Int(500),
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, 500, *err.StatusCode)

	err = NewSDKError(map[string]interface{}{
		"data": map[string]interface{}{
			"statusCode": "500",
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, 500, *err.StatusCode)

	err = NewSDKError(map[string]interface{}{
		"code":    "code",
		"message": "message",
		"data": map[string]interface{}{
			"requestId": "dfadfa32cgfdcasd4313",
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertNil(t, err.StatusCode)

	err = NewSDKError(map[string]interface{}{
		"code":    400,
		"message": "message",
		"data":    "string data",
	})
	utils.AssertNotNil(t, err)
	utils.AssertNotNil(t, err.Data)
	utils.AssertEqual(t, "400", StringValue(err.Code))
	utils.AssertNil(t, err.StatusCode)
}

func TestSDKErrorCode404(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"statusCode": 404,
		"code":       "NOTFOUND",
		"message":    "message",
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError:\n   StatusCode: 404\n   Code: NOTFOUND\n   Message: message\n   Data: \n", err.Error())
	utils.AssertEqual(t, "NOTFOUND", StringValue(err.ErrorCode()))
	utils.AssertEqual(t, "", StringValue(err.ErrorName()))
}

func TestNewError(t *testing.T) {
	var err0 BaseError
	err0 = NewError("ResponseError", map[string]interface{}{
		"code":       "code",
		"statusCode": 404,
		"message":    "message",
		"data": map[string]interface{}{
			"httpCode":  "404",
			"requestId": "dfadfa32cgfdcasd4313",
			"hostId":    "github.com/alibabacloud/tea",
			"recommend": "https://中文?q=a.b&product=c&requestId=123",
		},
		"description": "description",
		"accessDeniedDetail": map[string]interface{}{
			"AuthAction":        "ram:ListUsers",
			"AuthPrincipalType": "SubUser",
			"PolicyType":        "ResourceGroupLevelIdentityBassdPolicy",
			"NoPermissionType":  "ImplicitDeny",
			"UserId":            123,
		},
		"requestId":  "123456",
		"retryable":  true,
		"retryAfter": int64(100),
	})
	utils.AssertNotNil(t, err0)
	utils.AssertEqual(t, "SDKError:\n   StatusCode: 404\n   Code: code\n   Message: message\n   Data: {\"hostId\":\"github.com/alibabacloud/tea\",\"httpCode\":\"404\",\"recommend\":\"https://中文?q=a.b&product=c&requestId=123\",\"requestId\":\"dfadfa32cgfdcasd4313\"}\n", err0.Error())
	var err *SDKError
	err = err0.(*SDKError)
	err.SetErrMsg("test")
	utils.AssertEqual(t, "test", err.Error())
	utils.AssertEqual(t, 404, *err.StatusCode)
	utils.AssertEqual(t, "description", *err.Description)
	utils.AssertEqual(t, "ImplicitDeny", err.AccessDeniedDetail["NoPermissionType"])
	utils.AssertEqual(t, 123, err.AccessDeniedDetail["UserId"])
}
