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
	shifted := modifiers.Shift

	switch keycode {
	case 1:
		return KeysymEscape
	case 14:
		return KeysymBackSpace
	case 15:
		return KeysymTab
	case 28:
		return KeysymReturn
	case 42, 54:
		if shifted {
			return KeysymShiftR
		}
		return KeysymShiftL
	case 29, 97:
		if shifted {
			return KeysymControlR
		}
		return KeysymControlL
	case 56, 100:
		if shifted {
			return KeysymAltR
		}
		return KeysymAltL
	case 102:
		return KeysymHome
	case 103:
		return KeysymUp
	case 104:
		return KeysymPageUp
	case 105:
		return KeysymLeft
	case 106:
		return KeysymRight
	case 107:
		return KeysymEnd
	case 108:
		return KeysymDown
	case 109:
		return KeysymPageDown
	case 111:
		return KeysymDelete
	case 125, 126:
		if shifted {
			return KeysymSuperR
		}
		return KeysymSuperL
	}

	return km.keycodeToAlphanumeric(keycode, shifted)
}

func (km *Keymap) keycodeToAlphanumeric(keycode uint32, shifted bool) Keysym {
	// Keycode 2 = '1', 3 = '2', ..., 10 = '9', 11 = '0'
	if keycode >= 2 && keycode <= 11 {
		if !shifted {
			if keycode == 11 {
				return Keysym('0')
			}
			return Keysym('0' + (keycode - 1))
		}
		// Shifted: 1→!, 2→@, 3→#, 4→$, 5→%, 6→^, 7→&, 8→*, 9→(, 0→)
		shiftedDigits := "!@#$%^&*()"
		return Keysym(shiftedDigits[keycode-2])
	}

	if keycode >= 16 && keycode <= 25 {
		chars := "qwertyuiop"
		if shifted {
			return Keysym(chars[keycode-16] - 32)
		}
		return Keysym(chars[keycode-16])
	}

	if keycode >= 30 && keycode <= 38 {
		chars := "asdfghjkl"
		if shifted {
			return Keysym(chars[keycode-30] - 32)
		}
		return Keysym(chars[keycode-30])
	}

	if keycode >= 44 && keycode <= 50 {
		chars := "zxcvbnm"
		if shifted {
			return Keysym(chars[keycode-44] - 32)
		}
		return Keysym(chars[keycode-44])
	}

	if keycode == 57 {
		return Keysym(' ')
	}

	return 0
}

// Close releases resources associated with the keymap.
func (km *Keymap) Close() error {
	if km.data != nil {
		return syscall.Munmap(km.data)
	}
	return nil
}
