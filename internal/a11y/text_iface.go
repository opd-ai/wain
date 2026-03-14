package a11y

import (
	"github.com/godbus/dbus/v5"
)

// textIface exports org.a11y.atspi.Text for an AccessibleObject.
// Text provides access to the textual content and caret state of Entry widgets.
type textIface struct{ obj *AccessibleObject }

// clampOffsets ensures start and end are within [0, len(text)].
func clampOffsets(text string, start, end int32) (int, int) {
	n := int32(len([]rune(text)))
	if start < 0 {
		start = 0
	}
	if end < 0 || end > n {
		end = n
	}
	if start > end {
		end = start
	}
	return int(start), int(end)
}

// GetCharacterCount returns the number of Unicode code points in the text.
func (t *textIface) GetCharacterCount() (int32, *dbus.Error) {
	s := t.obj.snap()
	return int32(len([]rune(s.text))), nil
}

// GetText returns the substring of text between startOffset and endOffset.
// Passing endOffset = -1 returns text from startOffset to end.
func (t *textIface) GetText(startOffset, endOffset int32) (string, *dbus.Error) {
	s := t.obj.snap()
	runes := []rune(s.text)
	from, to := clampOffsets(s.text, startOffset, endOffset)
	return string(runes[from:to]), nil
}

// GetCaret returns the current caret (cursor) position in Unicode code points.
func (t *textIface) GetCaret() (int32, *dbus.Error) {
	s := t.obj.snap()
	return s.caretOffset, nil
}

// SetCaret moves the caret to the given Unicode code point offset.
// Returns false if the offset is out of range.
func (t *textIface) SetCaret(offset int32) (bool, *dbus.Error) {
	runes := []rune(t.obj.snap().text)
	if offset < 0 || int(offset) > len(runes) {
		return false, nil
	}
	t.obj.mu.Lock()
	t.obj.caretOffset = offset
	t.obj.mu.Unlock()
	return true, nil
}

// GetTextAfterOffset returns the word or line after the given offset.
// boundary: 0=char, 1=word-start, 2=word-end, 3=sentence-start, 4=sentence-end,
// 5=line-start, 6=line-end. Only char boundary (0) is implemented.
func (t *textIface) GetTextAfterOffset(offset int32, _ uint32) (string, int32, int32, *dbus.Error) {
	s := t.obj.snap()
	runes := []rune(s.text)
	n := int32(len(runes))
	if offset < 0 || offset >= n {
		return "", 0, 0, nil
	}
	next := offset + 1
	return string(runes[offset:next]), offset, next, nil
}

// GetTextAtOffset returns the character at the given offset.
func (t *textIface) GetTextAtOffset(offset int32, _ uint32) (string, int32, int32, *dbus.Error) {
	s := t.obj.snap()
	runes := []rune(s.text)
	n := int32(len(runes))
	if offset < 0 || offset >= n {
		return "", 0, 0, nil
	}
	return string(runes[offset : offset+1]), offset, offset + 1, nil
}

// GetTextBeforeOffset returns the character before the given offset.
func (t *textIface) GetTextBeforeOffset(offset int32, _ uint32) (string, int32, int32, *dbus.Error) {
	s := t.obj.snap()
	runes := []rune(s.text)
	if offset <= 0 || int(offset) > len(runes) {
		return "", 0, 0, nil
	}
	prev := offset - 1
	return string(runes[prev:offset]), prev, offset, nil
}

// GetDefaultAttributeSet returns an empty attribute map.
func (t *textIface) GetDefaultAttributeSet() (map[string]string, *dbus.Error) {
	return map[string]string{}, nil
}
