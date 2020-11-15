package main

import (
	"context"
	"github.com/hashicorp/consul/api"
	"time"
)

type options struct {
	SessionName string
	SessionTTL string
	SessionBehavior string
	LockDelayTime time.Duration
	LockWaitTime time.Duration
}

var defaultOptions = options{
	SessionName: api.DefaultLockSessionName,
	SessionTTL: api.DefaultLockSessionTTL,
	SessionBehavior: api.SessionBehaviorDelete,
}

type Option interface {
	apply(*options)
}

type funcOption struct {
	f func(*options)
}

func newFuncOption(f func(*options)) *funcOption {
	return &funcOption{
		f: f,
	}
}

func (fo *funcOption) apply(opts *options) {
	fo.f(opts)
}

func WithSessionName(name string) Option {
	return newFuncOption(func(o *options) {
		o.SessionName = name
	})
}

func WithSessionTTL(ttl time.Duration) Option {
	return newFuncOption(func(o *options) {
		o.SessionTTL = ttl.String()
	})
}

func WithSessionBehavior(behavior string) Option {
	return newFuncOption(func(o *options) {
		o.SessionBehavior = behavior
	})
}

func WithLockDelayTime(lockDelayTime time.Duration) Option {
	return newFuncOption(func(o *options) {
		o.LockDelayTime = lockDelayTime
	})
}

func WithLockWaitTime(lockWaitTime time.Duration) Option {
	return newFuncOption(func(o *options) {
		o.LockWaitTime = lockWaitTime
	})
}

type Lock struct {
	opts *options
	session *api.Session
	lock *api.Lock
	sessionId string
	sessionRenew chan struct{}
}

func NewLock(client *api.Client, key string, opt ...Option) (*Lock, error) {
	opts := defaultOptions
	for _, o := range opt {
		o.apply(&opts)
	}
	session := client.Session()
	se := &api.SessionEntry{
		Name:     opts.SessionName,
		TTL:      opts.SessionTTL,
		Behavior: opts.SessionBehavior,
		LockDelay: opts.LockDelayTime,
	}
	id, _, err := session.CreateNoChecks(se, nil)
	if err != nil {
		return nil, err
	}
	lockOpts := &api.LockOptions{
		Key:         key,
		Session:     id,
		SessionName: se.Name,
		SessionTTL:  se.TTL,
		LockWaitTime: opts.LockWaitTime,
	}
	lock, err := client.LockOpts(lockOpts)
	if err != nil  {
		return nil, err
	}
	l := &Lock{
		opts: &opts,
		session: session,
		lock: lock,
		sessionId: id,
		sessionRenew: make(chan struct{}),
	}
	go session.RenewPeriodic(opts.SessionTTL, id, nil, l.sessionRenew)
	return l, nil
}

func (l *Lock) Lock(ctx context.Context) (bool, error){
	stopCh := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(stopCh)
		}
	}()
	leaderCh, err := l.lock.Lock(stopCh)
	if err != nil {
		return false, err
	}
	if leaderCh == nil {
		return false, nil
	}
	return true, nil
}

func (l *Lock) Unlock() error {
	return l.lock.Unlock()
}

func (l *Lock) Destroy() error {
	defer l.session.Destroy(l.sessionId, nil)
	defer close(l.sessionRenew)
	return l.lock.Destroy()
}
