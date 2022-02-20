package types

import "context"

type Context struct {
	context.Context
}

type ContextOption func(c *Context)

func NewContext(options ...ContextOption) *Context {
	ctx := &Context{
		Context: context.Background(),
	}

	for _, opt := range options {
		opt(ctx)
	}
	return ctx
}

func WithCtx(ctx context.Context) ContextOption {
	return func(c *Context) {
		c.Context = ctx
	}
}
