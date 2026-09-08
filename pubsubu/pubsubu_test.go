package pubsubu

import (
	"sync"
	"testing"
)

func TestZeroValueUsable(t *testing.T) {
	var ps PubSub[int]

	sub := ps.Subscribe(nil)
	defer sub.Unsubscribe()

	if cap(sub.Ch) != DefaultChanBuffer {
		t.Fatalf("cap(sub.Ch) = %d, want %d", cap(sub.Ch), DefaultChanBuffer)
	}
	if sub.InitialValueValid {
		t.Fatal("InitialValueValid = true, want false")
	}

	ps.Notify(7)

	if got := <-sub.Ch; got != 7 {
		t.Fatalf("received %d, want 7", got)
	}
	if got := ps.Value(); got != 7 {
		t.Fatalf("Value() = %d, want 7", got)
	}
}

func TestValueOK(t *testing.T) {
	var ps PubSub[string]

	if v, ok := ps.ValueOK(); ok || v != "" {
		t.Fatalf("ValueOK() = %q, %v; want \"\", false", v, ok)
	}
	ps.Notify("")
	if v, ok := ps.ValueOK(); !ok || v != "" {
		t.Fatalf("ValueOK() = %q, %v; want \"\", true", v, ok)
	}
}

func TestNewPubSubExistingValue(t *testing.T) {
	ps := NewPubSub("a", 2)

	if v, ok := ps.ValueOK(); !ok || v != "a" {
		t.Fatalf("ValueOK() = %q, %v; want \"a\", true", v, ok)
	}

	sub := ps.Subscribe(func(old, new string) bool { return old != new })
	defer sub.Unsubscribe()

	if !sub.InitialValueValid || sub.InitialValue != "a" {
		t.Fatalf("InitialValue = %q, %v; want \"a\", true", sub.InitialValue, sub.InitialValueValid)
	}

	ps.Notify("a")
	if len(sub.Ch) != 0 {
		t.Fatalf("len(sub.Ch) = %d, want 0", len(sub.Ch))
	}

	ps.Notify("b")
	if got := <-sub.Ch; got != "b" {
		t.Fatalf("received %q, want \"b\"", got)
	}

	if got := ps.UpdateAndNotify(func(existing string) string { return existing + "c" }); got != "bc" {
		t.Fatalf("UpdateAndNotify returned %q, want \"bc\"", got)
	}
}

func TestSubscribeInitialValue(t *testing.T) {
	ps := NewPubSub(0, 4)
	ps.Notify(3)

	sub := ps.Subscribe(nil)
	defer sub.Unsubscribe()

	if !sub.InitialValueValid || sub.InitialValue != 3 {
		t.Fatalf("InitialValue = %d, %v; want 3, true", sub.InitialValue, sub.InitialValueValid)
	}
	if len(sub.Ch) != 0 {
		t.Fatalf("len(sub.Ch) = %d, want 0", len(sub.Ch))
	}
}

func TestFilterSeesPreviousValue(t *testing.T) {
	ps := NewPubSub(0, 4)

	var pairs [][2]int
	sub := ps.Subscribe(func(old, new int) bool {
		pairs = append(pairs, [2]int{old, new})
		return new%2 == 0
	})
	defer sub.Unsubscribe()

	ps.Notify(1)
	ps.Notify(2)
	ps.Notify(3)
	ps.Notify(4)

	want := [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 4}}
	if len(pairs) != len(want) {
		t.Fatalf("filter called %d times, want %d", len(pairs), len(want))
	}
	for i := range want {
		if pairs[i] != want[i] {
			t.Fatalf("filter call %d = %v, want %v", i, pairs[i], want[i])
		}
	}

	var got []int
	for len(sub.Ch) > 0 {
		got = append(got, <-sub.Ch)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("received %v, want [2 4]", got)
	}
}

func TestUpdateAndNotify(t *testing.T) {
	ps := NewPubSub(0, 4)

	sub := ps.Subscribe(nil)
	defer sub.Unsubscribe()

	if got := ps.UpdateAndNotify(func(existing int) int { return existing + 5 }); got != 5 {
		t.Fatalf("UpdateAndNotify returned %d, want 5", got)
	}
	if got := ps.UpdateAndNotify(func(existing int) int { return existing * 3 }); got != 15 {
		t.Fatalf("UpdateAndNotify returned %d, want 15", got)
	}
	if got := ps.Value(); got != 15 {
		t.Fatalf("Value() = %d, want 15", got)
	}
	if got := <-sub.Ch; got != 5 {
		t.Fatalf("received %d, want 5", got)
	}
	if got := <-sub.Ch; got != 15 {
		t.Fatalf("received %d, want 15", got)
	}
}

func TestUpdateAndNotifyConcurrent(t *testing.T) {
	ps := NewPubSub(0, 0)

	const goroutines, increments = 8, 500

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range increments {
				ps.UpdateAndNotify(func(existing int) int { return existing + 1 })
			}
		}()
	}
	wg.Wait()

	if got := ps.Value(); got != goroutines*increments {
		t.Fatalf("Value() = %d, want %d", got, goroutines*increments)
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	ps := NewPubSub(0, 4)

	a := ps.Subscribe(nil)
	b := ps.Subscribe(nil)
	defer b.Unsubscribe()

	ps.Notify(1)
	a.Unsubscribe()
	ps.Notify(2)

	var got []int
	for v := range a.Ch {
		got = append(got, v)
	}
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("a received %v, want [1]", got)
	}
	if len(b.Ch) != 2 {
		t.Fatalf("len(b.Ch) = %d, want 2", len(b.Ch))
	}
}

func TestUnsubscribeTwice(t *testing.T) {
	ps := NewPubSub(0, 1)
	sub := ps.Subscribe(nil)
	sub.Unsubscribe()
	sub.Unsubscribe()
}

func TestClosesWhenChannelFull(t *testing.T) {
	ps := NewPubSub(0, 1)
	sub := ps.Subscribe(nil)
	defer sub.Unsubscribe()

	ps.Notify(1)
	ps.Notify(2)

	if len(sub.Ch) != 1 {
		t.Fatalf("len(sub.Ch) = %d, want 1", len(sub.Ch))
	}
	if got := <-sub.Ch; got != 1 {
		t.Fatalf("received %d, want 1", got)
	}
	if _, ok := <-sub.Ch; ok {
		t.Fatal("overflowed subscriber is still open")
	}
	ps.Notify(3)
	sub.Unsubscribe()
	if got := ps.Value(); got != 3 {
		t.Fatalf("Value() = %d, want 3", got)
	}
}

func TestOnFullCalledWhenChannelFull(t *testing.T) {
	ps := NewPubSub(0, 1)

	var calls int
	// Unsubscribing from the callback is the point of running it outside the
	// lock, so the test does it rather than merely counting.
	var sub *Sub[int]
	sub = ps.Subscribe(nil, func() {
		calls++
		sub.Unsubscribe()
	})

	ps.Notify(1)
	if calls != 0 {
		t.Fatalf("onFull called %d times before the channel filled", calls)
	}

	ps.Notify(2)
	if calls != 1 {
		t.Fatalf("onFull called %d times, want 1", calls)
	}

	ps.Notify(3)
	if calls != 1 {
		t.Fatalf("onFull called %d times after unsubscribing, want 1", calls)
	}
}

func TestOnFullOptional(t *testing.T) {
	ps := NewPubSub(0, 1)
	sub := ps.Subscribe(nil)
	defer sub.Unsubscribe()

	ps.Notify(1)
	ps.Notify(2)
}

func TestNilPubSubNotify(t *testing.T) {
	var ps *PubSub[int]
	ps.Notify(1)
	if got := ps.UpdateAndNotify(func(existing int) int { return existing + 1 }); got != 0 {
		t.Fatalf("UpdateAndNotify returned %d, want 0", got)
	}
}

func TestConcurrentSubscribeNotifyUnsubscribe(t *testing.T) {
	ps := NewPubSub(0, 8)

	var producers, consumers sync.WaitGroup
	stop := make(chan struct{})

	for range 4 {
		producers.Add(1)
		go func() {
			defer producers.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				ps.Notify(i)
			}
		}()
	}

	for range 4 {
		consumers.Add(1)
		go func() {
			defer consumers.Done()
			for range 200 {
				sub := ps.Subscribe(func(old, new int) bool { return old != new })
				done := make(chan struct{})
				go func() {
					defer close(done)
					for range sub.Ch {
					}
				}()
				_ = ps.Value()
				ps.UpdateAndNotify(func(existing int) int { return existing + 1 })
				sub.Unsubscribe()
				<-done
			}
		}()
	}

	consumers.Wait()
	close(stop)
	producers.Wait()
}
