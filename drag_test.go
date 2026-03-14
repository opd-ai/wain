package wain

import (
	"testing"
)

// TestDragDropHandlerReceivesData verifies that dispatchDragEvent passes the
// mimeType and data fields of a DragDrop event to the registered handler.
func TestDragDropHandlerReceivesData(t *testing.T) {
	const wantMime = "text/plain"
	wantData := []byte("hello, drop")

	w := &Window{}

	var gotMime string
	var gotData []byte
	w.dropHandler = func(mime string, data []byte) {
		gotMime = mime
		gotData = data
	}

	app := &App{}
	evt := newDragDropEvent(10, 20, wantMime, wantData)
	app.dispatchDragEvent(w, evt)

	if gotMime != wantMime {
		t.Errorf("handler received mimeType=%q, want %q", gotMime, wantMime)
	}
	if string(gotData) != string(wantData) {
		t.Errorf("handler received data=%q, want %q", gotData, wantData)
	}
}

// TestDragDropHandlerEmptyWhenNoMimeMatch verifies that when no MIME type
// matches, the handler is still called (with empty strings) rather than
// silently skipped — so the caller can detect the failure mode.
func TestDragDropHandlerCalledWithEmptyOnNoMatch(t *testing.T) {
	called := false
	w := &Window{}
	w.dropHandler = func(mime string, data []byte) {
		called = true
	}

	app := &App{}
	evt := newDragDropEvent(0, 0, "", nil)
	app.dispatchDragEvent(w, evt)

	if !called {
		t.Error("expected drop handler to be called even with empty mimeType")
	}
}

// TestNegotiateDragMime verifies MIME type negotiation logic.
func TestNegotiateDragMime(t *testing.T) {
	tests := []struct {
		accepted []string
		offered  []string
		want     string
	}{
		{[]string{"text/plain"}, []string{"text/plain"}, "text/plain"},
		{[]string{"text/plain"}, []string{"image/png"}, ""},
		{[]string{"text/plain", "text/html"}, []string{"text/html"}, "text/html"},
		{nil, []string{"text/plain"}, ""},
		{[]string{"text/plain"}, nil, ""},
	}

	for _, tt := range tests {
		got := negotiateDragMime(tt.accepted, tt.offered)
		if got != tt.want {
			t.Errorf("negotiateDragMime(%v, %v) = %q, want %q", tt.accepted, tt.offered, got, tt.want)
		}
	}
}

// TestDragEventAccessors verifies new MimeType() and Data() accessors.
func TestDragEventAccessors(t *testing.T) {
	const mime = "application/octet-stream"
	data := []byte{1, 2, 3}

	evt := newDragDropEvent(5, 7, mime, data)

	if evt.MimeType() != mime {
		t.Errorf("MimeType() = %q, want %q", evt.MimeType(), mime)
	}
	if string(evt.Data()) != string(data) {
		t.Errorf("Data() = %v, want %v", evt.Data(), data)
	}
	if evt.Kind() != DragDrop {
		t.Errorf("Kind() = %v, want DragDrop", evt.Kind())
	}
	if evt.X() != 5 || evt.Y() != 7 {
		t.Errorf("X()=%v Y()=%v, want 5 7", evt.X(), evt.Y())
	}
}
