package u22

import "github.com/google/uuid"

//nolint:recvcheck // JSON Marshal/Unmarshal
type EncodedID uuid.UUID

//goland:noinspection GoMixedReceiverTypes
func (e EncodedID) String() string {
	return Encode(uuid.UUID(e))
}

//goland:noinspection GoMixedReceiverTypes
func (e EncodedID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + Encode(uuid.UUID(e)) + `"`), nil
}

//goland:noinspection GoMixedReceiverTypes
func (e *EncodedID) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return nil
	}
	if data[0] != '"' || data[len(data)-1] != '"' {
		return nil
	}
	id, err := Decode(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*e = EncodedID(id)
	return nil
}

func ConvertNullableID(id *uuid.UUID) *EncodedID {
	if id == nil {
		return nil
	}
	return (*EncodedID)(id)
}

func ToUUIDSlice(ids []EncodedID) []uuid.UUID {
	uuids := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = uuid.UUID(id)
	}
	return uuids
}

func FromUUIDSlice(ids []uuid.UUID) []EncodedID {
	encodedIDs := make([]EncodedID, len(ids))
	for i, id := range ids {
		encodedIDs[i] = EncodedID(id)
	}
	return encodedIDs
}
