package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Connection is the minimal interface needed for extension queries.
type Connection interface {
	ExtensionOpcode(name string) (uint8, error)
	SendRequestAndReply(data []byte) ([]byte, error)
}

// ExtensionInfo holds basic information about an X11 extension.
type ExtensionInfo struct {
	BaseOpcode   uint8
	MajorVersion uint32
	MinorVersion uint32
}

// QueryExtensionVersion queries an X11 extension by name and returns its opcode and version.
// This implements the common QueryVersion request pattern used by most extensions.
//
// Parameters:
//   - conn: X11 connection
//   - extensionName: Name of the extension (e.g., "DRI3", "Present")
//   - queryVersionOpcode: Extension-relative opcode for QueryVersion (usually 0)
//   - clientMajor: Client's major version
//   - clientMinor: Client's minor version
//
// Returns extension information or an error if the extension is not supported.
func QueryExtensionVersion(conn Connection, extensionName string, queryVersionOpcode uint8, clientMajor, clientMinor uint32) (*ExtensionInfo, error) {
	// Get extension opcode
	baseOpcode, err := conn.ExtensionOpcode(extensionName)
	if err != nil {
		return nil, fmt.Errorf("extension %q not supported: %w", extensionName, err)
	}

	// Build QueryVersion request
	var buf bytes.Buffer
	_ = EncodeRequestHeader(&buf, baseOpcode+queryVersionOpcode, 0, 3)
	_ = EncodeUint32(&buf, clientMajor)
	_ = EncodeUint32(&buf, clientMinor)

	// Send request and get reply
	reply, err := conn.SendRequestAndReply(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%s: QueryVersion failed: %w", extensionName, err)
	}

	// Parse reply: type(1) + pad(1) + sequence(2) + length(4) + major(4) + minor(4) + pad(16)
	if len(reply) < 32 {
		return nil, fmt.Errorf("%s: invalid QueryVersion reply (got %d bytes, expected ≥32)", extensionName, len(reply))
	}

	info := &ExtensionInfo{
		BaseOpcode:   baseOpcode,
		MajorVersion: binary.LittleEndian.Uint32(reply[8:12]),
		MinorVersion: binary.LittleEndian.Uint32(reply[12:16]),
	}

	return info, nil
}
