package tea

import (
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

type test struct {
	Key string `json:"key"`
}

func TestConvert(t *testing.T) {
	in := map[string]interface{}{
		"key": "value",
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "value", out.Key)
}

func TestConvertType(t *testing.T) {
	in := map[string]interface{}{
		"key": 123,
	}
	out := new(test)
	err := Convert(in, &out)
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "json: cannot unmarshal number into Go struct field test.key of type string", err.Error())
}

func TestSDKError(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":    "code",
		"message": "message",
		"data": map[string]interface{}{
			"httpCode":  "404",
			"requestId": "dfadfa32cgfdcasd4313",
			"hostId":    "github.com/alibabacloud/tea",
		},
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError: {\"hostId\":\"github.com/alibabacloud/tea\",\"httpCode\":\"404\",\"requestId\":\"dfadfa32cgfdcasd4313\"} message ", err.Error())
}

func TestSDKErrorCode404(t *testing.T) {
	err := NewSDKError(map[string]interface{}{
		"code":    404,
		"message": "message",
	})
	utils.AssertNotNil(t, err)
	utils.AssertEqual(t, "SDKError: 404 message ", err.Error())
}
