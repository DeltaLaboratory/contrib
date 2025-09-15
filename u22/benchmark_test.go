package u22

import (
	"testing"

	"github.com/google/uuid"
)

func BenchmarkEncode(b *testing.B) {
	u := uuid.New()
	b.ResetTimer()

	for b.Loop() {
		_ = Encode(u)
	}
}

func BenchmarkDecode(b *testing.B) {
	u := uuid.New()
	encoded := Encode(u)
	b.ResetTimer()

	for b.Loop() {
		_, _ = Decode(encoded)
	}
}

func BenchmarkEncodedIDString(b *testing.B) {
	id := EncodedID(uuid.New())
	b.ResetTimer()

	for b.Loop() {
		_ = id.String()
	}
}

func BenchmarkEncodedIDMarshalJSON(b *testing.B) {
	id := EncodedID(uuid.New())
	b.ResetTimer()

	for b.Loop() {
		_, _ = id.MarshalJSON()
	}
}

func BenchmarkEncodedIDUnmarshalJSON(b *testing.B) {
	id := EncodedID(uuid.New())
	data, _ := id.MarshalJSON()
	b.ResetTimer()

	var newID EncodedID
	for b.Loop() {
		_ = newID.UnmarshalJSON(data)
	}
}

func BenchmarkToUUIDSlice(b *testing.B) {
	ids := make([]EncodedID, 1000)
	for i := range ids {
		ids[i] = EncodedID(uuid.New())
	}
	b.ResetTimer()

	for b.Loop() {
		_ = ToUUIDSlice(ids)
	}
}

func BenchmarkFromUUIDSlice(b *testing.B) {
	uuids := make([]uuid.UUID, 1000)
	for i := range uuids {
		uuids[i] = uuid.New()
	}
	b.ResetTimer()

	for b.Loop() {
		_ = FromUUIDSlice(uuids)
	}
}

func BenchmarkUUIDStringVsU22String(b *testing.B) {
	u := uuid.New()

	b.Run("UUID.String()", func(b *testing.B) {
		for b.Loop() {
			_ = u.String()
		}
	})

	b.Run("U22.Encode()", func(b *testing.B) {
		for b.Loop() {
			_ = Encode(u)
		}
	})
}

func BenchmarkParseVsDecode(b *testing.B) {
	u := uuid.New()
	uuidStr := u.String()
	u22Str := Encode(u)

	b.Run("uuid.Parse()", func(b *testing.B) {
		for b.Loop() {
			_, _ = uuid.Parse(uuidStr)
		}
	})

	b.Run("u22.Decode()", func(b *testing.B) {
		for b.Loop() {
			_, _ = Decode(u22Str)
		}
	})
}

func BenchmarkMemoryAllocation(b *testing.B) {
	u := uuid.New()

	b.Run("Encode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = Encode(u)
		}
	})

	b.Run("Decode", func(b *testing.B) {
		encoded := Encode(u)
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_, _ = Decode(encoded)
		}
	})
}
