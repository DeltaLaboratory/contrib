package u22

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestEncodedIDString(t *testing.T) {
	tests := []struct {
		name string
		uuid uuid.UUID
	}{
		{
			name: "nil UUID",
			uuid: uuid.Nil,
		},
		{
			name: "random UUID",
			uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encodedID := EncodedID(tt.uuid)
			expected := Encode(tt.uuid)
			if encodedID.String() != expected {
				t.Errorf("String() = %v, want %v", encodedID.String(), expected)
			}
		})
	}
}

func TestEncodedIDMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		uuid uuid.UUID
	}{
		{
			name: "nil UUID",
			uuid: uuid.Nil,
		},
		{
			name: "random UUID",
			uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encodedID := EncodedID(tt.uuid)
			data, err := encodedID.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			expected := `"` + Encode(tt.uuid) + `"`
			if string(data) != expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(data), expected)
			}
		})
	}
}

func TestEncodedIDUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uuid.UUID
		wantErr bool
	}{
		{
			name:  "valid encoded UUID",
			input: `"` + Encode(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")) + `"`,
			want:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:    "empty string",
			input:   `""`,
			wantErr: true,
		},
		{
			name:    "short input",
			input:   `"a"`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `invalid`,
			wantErr: false,
		},
		{
			name:    "missing quotes",
			input:   `abcd`,
			wantErr: false,
		},
		{
			name:    "invalid encoded string",
			input:   `"invalid!"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var encodedID EncodedID
			err := encodedID.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && uuid.UUID(encodedID) != tt.want {
				t.Errorf("UnmarshalJSON() = %v, want %v", uuid.UUID(encodedID), tt.want)
			}
		})
	}
}

func TestEncodedIDJSONRoundtrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		original := EncodedID(uuid.New())

		data, err := json.Marshal(original)
		if err != nil {
			t.Errorf("iteration %d: Marshal() error = %v", i, err)
			continue
		}

		var decoded EncodedID
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Errorf("iteration %d: Unmarshal() error = %v", i, err)
			continue
		}

		if decoded != original {
			t.Errorf("iteration %d: roundtrip failed: %v != %v", i, decoded, original)
		}
	}
}

func TestConvertNullableID(t *testing.T) {
	tests := []struct {
		name string
		id   *uuid.UUID
		want *EncodedID
	}{
		{
			name: "nil UUID",
			id:   nil,
			want: nil,
		},
		{
			name: "valid UUID",
			id:   &[]uuid.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
			want: &[]EncodedID{EncodedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))}[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertNullableID(tt.id)
			if (got == nil) != (tt.want == nil) {
				t.Errorf("ConvertNullableID() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Errorf("ConvertNullableID() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestToUUIDSlice(t *testing.T) {
	tests := []struct {
		name string
		ids  []EncodedID
		want []uuid.UUID
	}{
		{
			name: "empty slice",
			ids:  []EncodedID{},
			want: []uuid.UUID{},
		},
		{
			name: "single ID",
			ids:  []EncodedID{EncodedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))},
			want: []uuid.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
		},
		{
			name: "multiple IDs",
			ids: []EncodedID{
				EncodedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
				EncodedID(uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")),
			},
			want: []uuid.UUID{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToUUIDSlice(tt.ids)
			if len(got) != len(tt.want) {
				t.Errorf("ToUUIDSlice() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, id := range got {
				if id != tt.want[i] {
					t.Errorf("ToUUIDSlice()[%d] = %v, want %v", i, id, tt.want[i])
				}
			}
		})
	}
}

func TestFromUUIDSlice(t *testing.T) {
	tests := []struct {
		name string
		ids  []uuid.UUID
		want []EncodedID
	}{
		{
			name: "empty slice",
			ids:  []uuid.UUID{},
			want: []EncodedID{},
		},
		{
			name: "single ID",
			ids:  []uuid.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
			want: []EncodedID{EncodedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))},
		},
		{
			name: "multiple IDs",
			ids: []uuid.UUID{
				uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			},
			want: []EncodedID{
				EncodedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
				EncodedID(uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromUUIDSlice(tt.ids)
			if len(got) != len(tt.want) {
				t.Errorf("FromUUIDSlice() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, id := range got {
				if id != tt.want[i] {
					t.Errorf("FromUUIDSlice()[%d] = %v, want %v", i, id, tt.want[i])
				}
			}
		})
	}
}
