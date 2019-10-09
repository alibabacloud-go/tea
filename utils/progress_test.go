package utils

import (
	"testing"
)

type Progresstest struct {
}

func (progress *Progresstest) ProgressChanged(event *ProgressEvent) {
}

func Test_PublishProgress(t *testing.T) {
	event := NewProgressEvent(TransferStartedEvent, int64(10), int64(20), int64(10))
	listener := GetProgressListener(nil)
	AssertNil(t, listener)

	listener = GetProgressListener(&testing.T{})
	AssertNil(t, listener)

	listener = GetProgressListener(&Progresstest{})
	PublishProgress(listener, event)
}
