package runlog_test

import (
	"context"
	"testing"
	"time"

	"github.com/efekckk/flowgent/internal/runlog"
)

func TestStreamer_PublishWithoutSubscribersNoPanic(t *testing.T) {
	s := runlog.New()
	s.Publish(context.Background(), runlog.Event{RunID: "r1", Message: "hi"})
}

func TestStreamer_SubscribeReceivesPublish(t *testing.T) {
	s := runlog.New()
	ch, unsub := s.Subscribe("r1")
	defer unsub()
	s.Publish(context.Background(), runlog.Event{RunID: "r1", Message: "hello"})
	select {
	case e := <-ch:
		if e.Message != "hello" {
			t.Errorf("got: %+v", e)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected event within 200ms")
	}
}

func TestStreamer_TwoSubscribersBothReceive(t *testing.T) {
	s := runlog.New()
	c1, u1 := s.Subscribe("r1")
	c2, u2 := s.Subscribe("r1")
	defer u1()
	defer u2()
	s.Publish(context.Background(), runlog.Event{RunID: "r1", Message: "bcast"})
	for _, ch := range []<-chan runlog.Event{c1, c2} {
		select {
		case e := <-ch:
			if e.Message != "bcast" {
				t.Errorf("got: %+v", e)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("expected event on each subscriber")
		}
	}
}

func TestStreamer_SubscribeIsolatedByRunID(t *testing.T) {
	s := runlog.New()
	cA, uA := s.Subscribe("rA")
	cB, uB := s.Subscribe("rB")
	defer uA()
	defer uB()

	s.Publish(context.Background(), runlog.Event{RunID: "rA", Message: "for A"})
	select {
	case e := <-cA:
		if e.Message != "for A" {
			t.Errorf("got: %+v", e)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected event on rA channel")
	}
	select {
	case e := <-cB:
		t.Errorf("rB should NOT receive rA event, got %+v", e)
	case <-time.After(100 * time.Millisecond):
		// ok
	}
}

func TestStreamer_SlowSubscriberDoesNotBlockEngine(t *testing.T) {
	s := runlog.New()
	_, unsub := s.Subscribe("r1")
	defer unsub()
	// Don't read from channel — fill its 32-slot buffer then publish more.
	for i := 0; i < 100; i++ {
		s.Publish(context.Background(), runlog.Event{RunID: "r1", Message: "x"})
	}
	// If Publish blocks we'd never reach here.
}

func TestStreamer_UnsubscribeClosesChannel(t *testing.T) {
	s := runlog.New()
	ch, unsub := s.Subscribe("r1")
	unsub()
	select {
	case _, ok := <-ch:
		if ok {
			t.Errorf("expected closed channel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected channel to be closed and read to return")
	}
	// Calling unsubscribe twice is safe (sync.Once).
	unsub()
}
