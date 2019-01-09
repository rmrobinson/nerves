package main

import (
	"github.com/rmrobinson/nerves/services/ui/tboard/widget"
)

// WidgetSink implements zap.Sink by writing all messages to a buffer.
type WidgetSink struct {
	widget *widget.Debug
}

// NewWidgetSink creates a new widget logger sink
func NewWidgetSink(widget *widget.Debug) *WidgetSink {
	return &WidgetSink{
		widget: widget,
	}
}

// Write saves the contents to the widget
func (s *WidgetSink) Write(p []byte) (n int, err error) {
	s.widget.Refresh(string(p))
	return len(p), nil
}

// Close is a nop
func (s *WidgetSink) Close() error { return nil }

// Sync is a nop
func (s *WidgetSink) Sync() error { return nil }
