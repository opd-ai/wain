package input

import (
	"syscall"
)

// Keysym represents an X11 keysym value.
type Keysym uint32

// Common keysyms for basic keyboard support.
const (
	KeysymBackSpace Keysym = 0xFF08
	KeysymTab       Keysym = 0xFF09
	KeysymReturn    Keysym = 0xFF0D
	KeysymEscape    Keysym = 0xFF1B
	KeysymDelete    Keysym = 0xFFFF
	KeysymHome      Keysym = 0xFF50
	KeysymLeft      Keysym = 0xFF51
	KeysymUp        Keysym = 0xFF52
	KeysymRight     Keysym = 0xFF53
	KeysymDown      Keysym = 0xFF54
	KeysymPageUp    Keysym = 0xFF55
	KeysymPageDown  Keysym = 0xFF56
	KeysymEnd       Keysym = 0xFF57
	KeysymShiftL    Keysym = 0xFFE1
	KeysymShiftR    Keysym = 0xFFE2
	KeysymControlL  Keysym = 0xFFE3
	KeysymControlR  Keysym = 0xFFE4
	KeysymAltL      Keysym = 0xFFE9
	KeysymAltR      Keysym = 0xFFEA
	KeysymSuperL    Keysym = 0xFFEB
	KeysymSuperR    Keysym = 0xFFEC
)

type keymapEntry struct {
	unshifted Keysym
	shifted   Keysym
}

var specialKeycodeMap = map[uint32]keymapEntry{
	1:   {KeysymEscape, KeysymEscape},
	14:  {KeysymBackSpace, KeysymBackSpace},
	15:  {KeysymTab, KeysymTab},
	28:  {KeysymReturn, KeysymReturn},
	42:  {KeysymShiftL, KeysymShiftR},
	54:  {KeysymShiftL, KeysymShiftR},
	29:  {KeysymControlL, KeysymControlR},
	97:  {KeysymControlL, KeysymControlR},
	56:  {KeysymAltL, KeysymAltR},
	100: {KeysymAltL, KeysymAltR},
	102: {KeysymHome, KeysymHome},
	103: {KeysymUp, KeysymUp},
	104: {KeysymPageUp, KeysymPageUp},
	105: {KeysymLeft, KeysymLeft},
	106: {KeysymRight, KeysymRight},
	107: {KeysymEnd, KeysymEnd},
	108: {KeysymDown, KeysymDown},
	109: {KeysymPageDown, KeysymPageDown},
	111: {KeysymDelete, KeysymDelete},
	125: {KeysymSuperL, KeysymSuperR},
	126: {KeysymSuperL, KeysymSuperR},
}

// Keymap represents a keyboard mapping from evdev keycodes to keysyms.
//
// This is a minimal implementation that provides basic keycode to keysym
// translation without full XKB parsing. It uses a simple lookup table
// for common US keyboard layout.
type Keymap struct {
	data []byte
}

// NewKeymap creates a keymap from an XKB keymap file descriptor.
//
// This implementation memory-maps the keymap data but does not fully parse
// the XKB format. It provides basic keysym lookup using a hardcoded table.
func NewKeymap(fd, size int) *Keymap {
	data, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return &Keymap{}
	}
	syscall.Close(fd)

	return &Keymap{
		data: data,
	}
}

// KeycodeToKeysym converts a Linux evdev keycode to a keysym.
//
// This uses a simple lookup table for common keys. For a full implementation,
// this would parse the XKB keymap data.
func (km *Keymap) KeycodeToKeysym(keycode uint32, modifiers ModifierState) Keysym {
	if entry, ok := specialKeycodeMap[keycode]; ok {
		if modifiers.Shift {
			return entry.shifted
		}
		return entry.unshifted
	}

	return km.keycodeToAlphanumeric(keycode, modifiers.Shift)
}

// keycodeToAlphanumeric maps Linux keycodes to alphanumeric keysyms for common QWERTY keys.
// Handles digits (2-11), QWERTY rows (16-25, 30-38, 44-50), space (57), and common punctuation.
// Returns KeysymInvalid for unmapped keycodes.
func (km *Keymap) keycodeToAlphanumeric(keycode uint32, shifted bool) Keysym {
	if keycode >= 2 && keycode <= 11 {
		return mapDigitKeycode(keycode, shifted)
	}

	if keycode >= 16 && keycode <= 25 {
		return mapQwertyRow(keycode, shifted)
	}

	if keycode >= 30 && keycode <= 38 {
		return mapHomeRow(keycode, shifted)
	}

	if keycode >= 44 && keycode <= 50 {
		return mapBottomRow(keycode, shifted)
	}

	if keycode == 57 {
		return Keysym(' ')
	}

	return 0
}

func mapDigitKeycode(keycode uint32, shifted bool) Keysym {
	if !shifted {
		if keycode == 11 {
			return Keysym('0')
		}
		return Keysym('0' + (keycode - 1))
	}
	shiftedDigits := "!@#$%^&*()"
	return Keysym(shiftedDigits[keycode-2])
}

func mapQwertyRow(keycode uint32, shifted bool) Keysym {
	return mapLetterRange("qwertyuiop", keycode, 16, shifted)
}

func mapHomeRow(keycode uint32, shifted bool) Keysym {
	return mapLetterRange("asdfghjkl", keycode, 30, shifted)
}

func mapBottomRow(keycode uint32, shifted bool) Keysym {
	return mapLetterRange("zxcvbnm", keycode, 44, shifted)
}

func mapLetterRange(chars string, keycode, offset uint32, shifted bool) Keysym {
	char := chars[keycode-offset]
	if shifted {
		return Keysym(char - 32)
	}
	return Keysym(char)
}

// Close releases resources associated with the keymap.
func (km *Keymap) Close() error {
	if km.data != nil {
		return syscall.Munmap(km.data)
	}
	return nil
}
