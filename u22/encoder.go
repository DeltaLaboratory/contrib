package u22

import (
	"fmt"

	"github.com/google/uuid"
)

const base6Length = 22

var Charset = []byte("abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789" + "_-")

func Encode(u uuid.UUID) string {
	// Get raw bytes from UUID (16 bytes = 128 bits)
	uuidBytes := u[:]

	// Initialize result slice
	result := make([]byte, base6Length)

	// Track bit positions
	bitPos := 0
	bytePos := 0

	// Convert to base64
	for i := base6Length - 1; i >= 0; i-- {
		// Collect 6 bits
		var chunk byte
		for j := range 6 {
			if bytePos < len(uuidBytes) {
				bit := (uuidBytes[bytePos] >> bitPos) & 1
				chunk |= bit << j

				bitPos++
				if bitPos == 8 {
					bitPos = 0
					bytePos++
				}
			}
		}
		result[i] = Charset[chunk]
	}

	return string(result)
}

func Decode(encoded string) (uuid.UUID, error) {
	if len(encoded) != base6Length {
		return uuid.Nil, fmt.Errorf("invalid encoded length: %d", len(encoded))
	}

	// Create reverse lookup map
	dictMap := make(map[byte]byte, len(Charset))
	for i, c := range Charset {
		dictMap[c] = byte(i)
	}

	var uuidBytes [16]byte
	bitPos := 0
	bytePos := 0

	// Process each character from right to left
	for i := len(encoded) - 1; i >= 0; i-- {
		val, ok := dictMap[encoded[i]]
		if !ok {
			return uuid.Nil, fmt.Errorf("invalid character in encoded string: %c", encoded[i])
		}

		// Distribute 6 bits into UUID bytes
		for j := range 6 {
			if bytePos < len(uuidBytes) {
				bit := (val >> j) & 1
				uuidBytes[bytePos] |= bit << bitPos

				bitPos++
				if bitPos == 8 {
					bitPos = 0
					bytePos++
				}
			}
		}
	}

	return uuidBytes, nil
}
