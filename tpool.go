package gojsx

import (
	"context"
	pool "github.com/jolestar/go-commons-pool/v2"
)

type tPool[T any] struct {
	op *pool.ObjectPool
}

func newTPool[T any](maxTotal int, fun func() T) *tPool[T] {
	factory := pool.NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return fun(), nil
		})
	ctx := context.Background()
	p := pool.NewObjectPool(ctx, factory, &pool.ObjectPoolConfig{
		MaxIdle:            -1,
		MaxTotal:           maxTotal,
		BlockWhenExhausted: true,
	})

	return &tPool[T]{
		op: p,
	}
}

func (p *tPool[T]) Get() (t T, err error) {
	o, err := p.op.BorrowObject(context.Background())
	if err != nil {
		return
	}

	return o.(T), nil
}

func (p *tPool[T]) Put(t T) error {
	err := p.op.ReturnObject(context.Background(), t)
	if err != nil {
		return err
	}
	return nil
}
