package u22

import (
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
)

const (
	encodeLength      = 22
	mask6             = 0x3F // 00111111
	invalid      byte = 0xFF
)

var (
	charset    = []byte("abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789" + "_-")
	charsetMap [256]byte
)

func init() {
	for i := range charsetMap {
		charsetMap[i] = invalid
	}
	for i, c := range charset {
		charsetMap[c] = byte(i)
	}
}

func Encode(u uuid.UUID) string {
	// Split the 16-byte UUID into two 64-bit integers (little-endian).
	// lo = u[0:8], hi = u[8:16]
	lo := binary.LittleEndian.Uint64(u[0:8])
	hi := binary.LittleEndian.Uint64(u[8:16])

	// Use a stack-allocated array instead of make() for performance.
	var result [encodeLength]byte

	// 1. Fill the 60 bits of lo (10 * 6) (right to left).
	// i = 21 ~ 12
	for i := encodeLength - 1; i >= encodeLength-10; i-- {
		result[i] = charset[lo&mask6]
		lo >>= 6
	}

	// 2. Fill the "boundary" character.
	// Remaining 4 bits of lo + first 2 bits of hi.
	// i = 11
	result[encodeLength-11] = charset[(lo&0x0F)|((hi&0x03)<<4)]
	hi >>= 2 // Remove the 2 bits used from hi.

	// 3. Fill the remaining 62 bits of hi (11 * 6 = 66 bits of space).
	// i = 10 ~ 0
	for i := encodeLength - 12; i >= 0; i-- {
		result[i] = charset[hi&mask6]
		hi >>= 6
	}

	return string(result[:])
}

func Decode(encoded string) (uuid.UUID, error) {
	if len(encoded) != encodeLength {
		return uuid.Nil, fmt.Errorf("invalid encoded length: %d", len(encoded))
	}

	var uuidBytes [16]byte
	var lo, hi uint64

	// 1. Build hi (encoded[0] ~ encoded[10]).
	// i = 0 ~ 10
	for i := 0; i < encodeLength-11; i++ {
		c := encoded[i]
		val := charsetMap[c]
		if val == invalid {
			return uuid.Nil, fmt.Errorf("invalid character in encoded string: %c", c)
		}
		hi = (hi << 6) | uint64(val)
	}

	// 2. Process the "boundary" character (encoded[11]).
	c := encoded[encodeLength-11]
	val := charsetMap[c]
	if val == invalid {
		return uuid.Nil, fmt.Errorf("invalid character in encoded string: %c", c)
	}

	// 3. Build lo (encoded[12] ~ encoded[21]).
	// i = 12 ~ 21
	for i := encodeLength - 10; i < encodeLength; i++ {
		c := encoded[i]
		v := charsetMap[c]
		if v == invalid {
			return uuid.Nil, fmt.Errorf("invalid character in encoded string: %c", c)
		}
		lo = (lo << 6) | uint64(v)
	}

	// 4. Distribute the boundary value (val) to hi and lo.
	// hi = (hi << 2) | (2 MSBs of val)
	hi = (hi << 2) | (uint64(val>>4) & 0x03)
	// lo = (60 LSBs of lo) | (4 LSBs of val shifted left by 60 bits)
	lo |= uint64(val&0x0F) << 60

	// 5. Convert two uint64 integers to a 16-byte UUID.
	binary.LittleEndian.PutUint64(uuidBytes[0:8], lo)
	binary.LittleEndian.PutUint64(uuidBytes[8:16], hi)

	return uuidBytes, nil
}
