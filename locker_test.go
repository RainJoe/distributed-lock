package main

import (
	"context"
	"github.com/hashicorp/consul/api"
	"testing"
	"time"
)

func TestLock_LockUnlock(t *testing.T) {
	t.Parallel()
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to get consul client: %v", err)
	}
	lock, err := NewLock(client, "test/lock")
	if err != nil {
		t.Fatalf("failed to create lock: %v", err)
	}
	defer lock.Destroy()
	// Initial lock should work
	locked, err := lock.Lock(context.TODO())
	if err != nil {
		t.Fatalf("failed to lock: %v", err)
	}
	if !locked {
		t.Error("initial lock should get true but get false")
	}

	// Double lock should fail
	locked, err = lock.Lock(context.TODO())
	if err != nil && err != api.ErrLockHeld{
		t.Fatalf("failed to lock: %v", err)
	}
	if locked {
		t.Error("initial lock should get false but get true")
	}

	// Initial unlock should work
	err = lock.Unlock()
	if err != nil {
		t.Fatalf("failed to unlock: %v", err)
	}
}

func TestWithLockDelayTimeLockWaitTime(t *testing.T) {
	t.Parallel()
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to get consul client: %v", err)
	}
	lock, err := NewLock(client, "test/lock", WithLockDelayTime(10 * time.Second), WithLockWaitTime(1 * time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create lock: %v", err)
	}
	defer lock.Destroy()
	// Initial lock should work
	locked, err := lock.Lock(context.TODO())
	if err != nil {
		t.Fatalf("failed to lock: %v", err)
	}
	if !locked {
		t.Error("initial lock should get true but get false")
	}
	go func() {
		// Nuke the session, simulator an operator invalidation
		// or a health check failure
		session := client.Session()
		session.Destroy(lock.sessionId, nil)
	}()
	select {
	case <- time.After(5 * time.Second):
		client, err = api.NewClient(api.DefaultConfig())
		if err != nil {
			t.Fatalf("failed to get consul client: %v", err)
		}
		lock, err := NewLock(client, "test/lock", WithLockDelayTime(10 * time.Second), WithLockWaitTime(1 * time.Millisecond))
		if err != nil {
			t.Fatalf("failed to create lock: %v", err)
		}
		// Should fail if get lock in lock delay time
		ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
		defer cancel()
		locked, err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("failed to lock: %v", err)
		}
		if locked {
			t.Error("initial lock should get false but get true")
		}
	case <-time.After(15 * time.Second):
		// Should work if get lock after lock delay time
		ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
		defer cancel()
		locked, err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("failed to lock: %v", err)
		}
		if !locked {
			t.Error("initial lock should get true but get false")
		}
	}
}