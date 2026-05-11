package eltest_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ttab/eltest"
)

func TestValkey(t *testing.T) {
	v := eltest.NewValkey(t, eltest.Valkey8_0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	client := v.Client()
	t.Cleanup(func() {
		_ = client.Close()
	})

	err := client.Set(ctx, "hello", "world", 0).Err()
	eltest.Must(t, err, "set key")

	got, err := client.Get(ctx, "hello").Result()
	eltest.Must(t, err, "get key")

	if got != "world" {
		t.Fatalf("got %q back, expected %q", got, "world")
	}

	// Streams round-trip: collab's primary use of valkey is XADD/XREAD,
	// so prove that path works end to end.
	streamID, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: "test-stream",
		Values: map[string]any{"data": "first"},
	}).Result()
	eltest.Must(t, err, "xadd")

	entries, err := client.XRange(ctx, "test-stream", "-", "+").Result()
	eltest.Must(t, err, "xrange")

	if len(entries) != 1 {
		t.Fatalf("XRange returned %d entries, want 1", len(entries))
	}
	if entries[0].ID != streamID {
		t.Errorf("entry ID = %q, want %q", entries[0].ID, streamID)
	}
	if entries[0].Values["data"] != "first" {
		t.Errorf("entry data = %v, want %q", entries[0].Values["data"], "first")
	}
}
