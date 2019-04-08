package tea_test

import (
	"testing"

	"github.com/aliyun/tea"
	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	in := map[string]interface{}{
		"key": "value",
	}
	out := &struct {
		Key string
	}{}
	err := tea.Convert(in, out)
	assert.Nil(t, err)
	assert.Equal(t, "value", out.Key)
}

func TestConvertNonPtr(t *testing.T) {
	in := map[string]interface{}{
		"key": "value",
	}
	out := struct {
		Key string
	}{}
	err := tea.Convert(in, out)
	assert.NotNil(t, err)
	assert.Equal(t, "The out parameter must be pointer", err.Error())
}

func TestConvertType(t *testing.T) {
	in := map[string]interface{}{
		"key": "value",
	}
	out := &struct {
		Key int
	}{}
	err := tea.Convert(in, out)
	assert.NotNil(t, err)
	assert.Equal(t, "Convert type fails for field: key, expect type: int, current type: string", err.Error())
}

func TestSDKError(t *testing.T) {
	err := tea.NewSDKError(map[string]interface{}{
		"code":    "code",
		"message": "message",
	})
	assert.NotNil(t, err)
	assert.Equal(t, "SDKError: code message", err.Error())
}

func TestSDKErrorCode404(t *testing.T) {
	err := tea.NewSDKError(map[string]interface{}{
		"code":    404,
		"message": "message",
	})
	assert.NotNil(t, err)
	assert.Equal(t, "SDKError: 404 message", err.Error())
}
