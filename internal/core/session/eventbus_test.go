package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEventBusKeepsThreadsIsolated(t *testing.T) {
	bus := NewEventBus(4)
	threadOne, cancelOne := bus.Subscribe("thread-1")
	defer cancelOne()
	threadTwo, cancelTwo := bus.Subscribe("thread-2")
	defer cancelTwo()

	dropped := bus.Publish("thread-1", Event{ID: "event-1", ThreadID: "thread-1", Type: "task.started"})
	require.False(t, dropped)

	select {
	case item := <-threadOne:
		require.Equal(t, "event-1", item.ID)
	case <-time.After(time.Second):
		t.Fatal("expected thread-1 event")
	}

	select {
	case item := <-threadTwo:
		t.Fatalf("unexpected thread-2 event: %#v", item)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEventBusDropsOldestWhenBufferIsFull(t *testing.T) {
	bus := NewEventBus(1)
	stream, cancel := bus.Subscribe("thread-1")
	defer cancel()

	require.False(t, bus.Publish("thread-1", Event{ID: "event-1"}))
	require.True(t, bus.Publish("thread-1", Event{ID: "event-2"}))

	select {
	case item := <-stream:
		require.Equal(t, "event-2", item.ID)
	case <-time.After(time.Second):
		t.Fatal("expected newest buffered event")
	}
}

func TestEventBusCancelStopsSubscription(t *testing.T) {
	bus := NewEventBus(1)
	stream, cancel := bus.Subscribe("thread-1")
	cancel()

	_, ok := <-stream
	require.False(t, ok)
}

