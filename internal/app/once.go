package app

import "sync"

// once is a generic lazy-init helper that captures both value and error from a
// single initialisation call. It replaces the pattern of pairing sync.Once with
// a sync.Map to propagate errors to concurrent callers.
//
// Thread safety: sync.Once guarantees that fn runs exactly once and that its
// side-effects are visible to all goroutines that call get after fn returns.
type once[T any] struct {
	sync.Once
	val T
	err error
}

func (o *once[T]) get(fn func() (T, error)) (T, error) {
	o.Do(func() { o.val, o.err = fn() })
	return o.val, o.err
}
