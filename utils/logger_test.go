package utils

import (
	"bytes"
	"errors"
	"testing"
)

func Test_Param(t *testing.T) {
	log := new(Logger)
	log.SetFormatTemplate("template")
	temp := log.GetFormatTemplate()
	AssertEqual(t, "template", temp)

	log.SetIsopen(true)
	isOpen := log.GetIsopen()
	AssertEqual(t, true, isOpen)

	log.SetLastLogMsg("logMsg")
	msg := log.GetLastLogMsg()
	AssertEqual(t, "logMsg", msg)

	log.CloseLogger()
	AssertEqual(t, false, log.isOpen)

	log.OpenLogger()
	AssertEqual(t, true, log.isOpen)
}

func Test_PrintLog(t *testing.T) {
	fieldMap := make(map[string]string)
	InitLogMsg(fieldMap)

	originlogChannel := logChannel
	SetLogChannel("Info")
	defer func() {
		logChannel = originlogChannel
	}()

	byt := new(bytes.Buffer)
	logger := NewLogger("", "tea", byt, "")
	logger.formatTemplate = "{channel} {error}"
	logger.SetOutput(byt)
	logger.PrintLog(fieldMap, errors.New("tea error"))
	AssertEqual(t, byt.String(), "[INFO]logger_test.go:44: tea tea error\n")
}
