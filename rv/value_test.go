package rv

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/redis/rueidis"
	rueidismock "github.com/redis/rueidis/mock"
	"go.uber.org/mock/gomock"
)

type testPayload struct {
	Message string
}

func TestValueSetAppliesDefaultExpiration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "default", WithDefaultExpiration(45*time.Second))

	client.EXPECT().
		Do(ctx, matchSetCommand("default:key", func(tokens []string) bool {
			return hasTokenSequence(tokens, "EX", secondsString(45*time.Second)) &&
				!containsToken(tokens, "KEEPTTL")
		})).
		Return(rueidismock.Result(rueidismock.RedisString("OK")))

	payload := testPayload{Message: "hello"}
	if err := value.Set(ctx, "key", &payload); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

func TestValueSetPrefersPerCallTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "percall", WithDefaultExpiration(2*time.Minute))

	perCallTTL := 15 * time.Second
	client.EXPECT().
		Do(ctx, matchSetCommand("percall:item", func(tokens []string) bool {
			return hasTokenSequence(tokens, "EX", secondsString(perCallTTL)) &&
				countToken(tokens, "EX") == 1 &&
				!containsToken(tokens, "KEEPTTL")
		})).
		Return(rueidismock.Result(rueidismock.RedisString("OK")))

	payload := testPayload{Message: "world"}
	if err := value.Set(ctx, "item", &payload, SetTTL(perCallTTL)); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

func TestValueSetKeepsExistingTTLWhenRequested(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "keep", WithDefaultExpiration(time.Minute))

	client.EXPECT().
		Do(ctx, matchSetCommand("keep:session", func(tokens []string) bool {
			return containsToken(tokens, "KEEPTTL") &&
				!containsToken(tokens, "EX")
		})).
		Return(rueidismock.Result(rueidismock.RedisString("OK")))

	payload := testPayload{Message: "keep"}
	if err := value.Set(ctx, "session", &payload, SetKeepTTL(true)); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

func TestValueSetRejectsConflictingTTLRequests(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "conflict", WithDefaultExpiration(time.Minute))

	payload := testPayload{Message: "oops"}
	err := value.Set(ctx, "key", &payload, SetTTL(time.Second), SetKeepTTL(true))
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	expected := "cannot use SetTTL and SetKeepTTL simultaneously"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValueSetEncodesPayload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "data")

	payload := testPayload{Message: "encoded"}
	encoded := string(mustEncode(payload))

	client.EXPECT().
		Do(ctx, matchSetCommand("data:record", func(tokens []string) bool {
			return len(tokens) >= 3 &&
				tokens[2] == encoded &&
				!containsToken(tokens, "EX") &&
				!containsToken(tokens, "KEEPTTL")
		})).
		Return(rueidismock.Result(rueidismock.RedisString("OK")))

	if err := value.Set(ctx, "record", &payload); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

func TestValueGetReturnsDecodedValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "get")

	payload := testPayload{Message: "from redis"}
	client.EXPECT().
		Do(ctx, matchGetCommand("get:item")).
		Return(rueidismock.Result(rueidismock.RedisBlobString(string(mustEncode(payload)))))

	result, err := value.Get(ctx, "item")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result == nil || result.Message != payload.Message {
		t.Fatalf("unexpected value: %#v", result)
	}
}

func TestValueGetReturnsNilWhenMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "missing")

	client.EXPECT().
		Do(ctx, matchGetCommand("missing:key")).
		Return(rueidismock.Result(rueidismock.RedisNil()))

	result, err := value.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
}

func TestValueDeleteRemovesKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "delete")

	client.EXPECT().
		Do(ctx, matchDeleteCommand("delete:key")).
		Return(rueidismock.Result(rueidismock.RedisInt64(1)))

	if err := value.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestValueScanReturnsDecodedValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := rueidismock.NewClient(ctrl)
	value := NewValue[testPayload](client, nil, "scan")

	firstKeys := []string{"scan:user:1", "scan:user:2"}
	matchPattern := "scan:user:*"
	firstValue := testPayload{Message: "user-1"}

	gomock.InOrder(
		client.EXPECT().
			Do(ctx, matchScanCommand(0, matchPattern)).
			Return(rueidismock.Result(scanResponse(1, firstKeys))),
		client.EXPECT().
			DoMulti(ctx,
				rueidismock.Match("GET", firstKeys[0]),
				rueidismock.Match("GET", firstKeys[1]),
			).
			Return([]rueidis.RedisResult{
				rueidismock.Result(rueidismock.RedisBlobString(string(mustEncode(firstValue)))),
				rueidismock.Result(rueidismock.RedisNil()),
			}),
		client.EXPECT().
			Do(ctx, matchScanCommand(1, matchPattern)).
			Return(rueidismock.Result(scanResponse(0, nil))),
	)

	values, err := value.Scan(ctx, "user:*")
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if values[0].Message != firstValue.Message {
		t.Fatalf("unexpected payload %+v", values[0])
	}
}
func matchSetCommand(key string, predicate func(tokens []string) bool) gomock.Matcher {
	desc := fmt.Sprintf("SET %s command", key)
	return rueidismock.MatchFn(func(tokens []string) bool {
		if len(tokens) < 3 {
			return false
		}
		if tokens[0] != "SET" || tokens[1] != key {
			return false
		}
		return predicate(tokens)
	}, desc)
}

func matchGetCommand(key string) gomock.Matcher {
	return rueidismock.Match("GET", key)
}

func matchDeleteCommand(key string) gomock.Matcher {
	return rueidismock.Match("DEL", key)
}

func matchScanCommand(cursor uint64, match string) gomock.Matcher {
	expectedCursor := strconv.FormatUint(cursor, 10)
	return rueidismock.MatchFn(func(tokens []string) bool {
		if len(tokens) < 4 {
			return false
		}
		return tokens[0] == "SCAN" &&
			tokens[1] == expectedCursor &&
			tokens[2] == "MATCH" &&
			tokens[3] == match
	}, fmt.Sprintf("SCAN cursor=%d match=%s", cursor, match))
}

func containsToken(tokens []string, needle string) bool {
	for _, token := range tokens {
		if token == needle {
			return true
		}
	}
	return false
}

func hasTokenSequence(tokens []string, token, next string) bool {
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i] == token && tokens[i+1] == next {
			return true
		}
	}
	return false
}

func countToken(tokens []string, needle string) int {
	var count int
	for _, token := range tokens {
		if token == needle {
			count++
		}
	}
	return count
}

func secondsString(d time.Duration) string {
	return strconv.FormatInt(int64(d/time.Second), 10)
}

func scanResponse(cursor uint64, keys []string) rueidis.RedisMessage {
	var elements []rueidis.RedisMessage
	for _, key := range keys {
		elements = append(elements, rueidismock.RedisString(key))
	}
	return rueidismock.RedisArray(
		rueidismock.RedisString(strconv.FormatUint(cursor, 10)),
		rueidismock.RedisArray(elements...),
	)
}

func mustEncode(payload testPayload) []byte {
	data, err := cbor.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return data
}
