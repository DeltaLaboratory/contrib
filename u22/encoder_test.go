package u22

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		uuid uuid.UUID
	}{
		{
			name: "nil UUID",
			uuid: uuid.Nil,
		},
		{
			name: "max UUID",
			uuid: uuid.Max,
		},
		{
			name: "random UUID",
			uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name: "another random UUID",
			uuid: uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := Encode(tt.uuid)
			if len(encoded) != encodeLength {
				t.Errorf("encoded length = %d, want %d", len(encoded), encodeLength)
			}

			decoded, err := Decode(encoded)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}

			if decoded != tt.uuid {
				t.Errorf("Decode(Encode(%v)) = %v, want %v", tt.uuid, decoded, tt.uuid)
			}
		})
	}
}

func TestEncodeDecodeKnownValues(t *testing.T) {
	tests := []struct {
		name    string
		uuid    uuid.UUID
		encoded string
	}{
		{
			name:    "nil UUID",
			uuid:    uuid.Nil,
			encoded: "aaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			name:    "max UUID",
			uuid:    uuid.Max,
			encoded: "d---------------------",
		},
		{
			name:    "random UUID 1",
			uuid:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			encoded: "aaaervzKqwP9rbM_iaHa5v",
		},
		{
			name:    "random UUID 2",
			uuid:    uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			encoded: "dimnrpWac0GnerRz0qUkDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Encode(tt.uuid); got != tt.encoded {
				t.Fatalf("Encode(%v) = %s, want %s", tt.uuid, got, tt.encoded)
			}

			decoded, err := Decode(tt.encoded)
			if err != nil {
				t.Fatalf("Decode(%s) error = %v", tt.encoded, err)
			}

			if decoded != tt.uuid {
				t.Fatalf("Decode(%s) = %v, want %v", tt.encoded, decoded, tt.uuid)
			}
		})
	}
}

func TestEncodeRandomUUIDs(t *testing.T) {
	for i := 0; i < 1000; i++ {
		original := uuid.New()
		encoded := Encode(original)
		decoded, err := Decode(encoded)
		if err != nil {
			t.Errorf("iteration %d: Decode() error = %v", i, err)
			continue
		}
		if decoded != original {
			t.Errorf("iteration %d: roundtrip failed: %v != %v", i, original, decoded)
		}
	}
}

func TestDecodeInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", encodeLength+1),
			wantErr: true,
		},
		{
			name:    "invalid character",
			input:   "invalid!" + strings.Repeat("a", encodeLength-8),
			wantErr: true,
		},
		{
			name:    "space character",
			input:   " " + strings.Repeat("a", encodeLength-1),
			wantErr: true,
		},
		{
			name:    "valid charset characters",
			input:   strings.Repeat("a", encodeLength),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeCharacterCaseSensitivity(t *testing.T) {
	u := uuid.New()
	encoded := Encode(u)

	lowerCaseEncoded := strings.ToLower(encoded)
	upperCaseEncoded := strings.ToUpper(encoded)

	if lowerCaseEncoded != encoded {
		decoded1, err1 := Decode(lowerCaseEncoded)
		if err1 != nil {
			t.Errorf("Decode lowercase failed: %v", err1)
			return
		}
		if decoded1 == u {
			t.Error("Case change should produce different UUID")
		}
	}

	if upperCaseEncoded != encoded {
		decoded2, err2 := Decode(upperCaseEncoded)
		if err2 != nil {
			t.Errorf("Decode uppercase failed: %v", err2)
			return
		}
		if decoded2 == u {
			t.Error("Case change should produce different UUID")
		}
	}
}
