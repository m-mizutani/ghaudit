package usecase

import (
	"path/filepath"

	"github.com/m-mizutani/ghaudit/pkg/infra"
)

type Usecase struct {
	clients *infra.Clients
	thread  int64
	limit   int64
	dumpDir string
}

func New(clients *infra.Clients, options ...Option) *Usecase {
	uc := &Usecase{
		clients: clients,
		thread:  4,
	}

	for _, opt := range options {
		opt(uc)
	}

	return uc
}

type Option func(uc *Usecase)

func WithThread(n int64) Option {
	return func(uc *Usecase) {
		uc.thread = n
	}
}

func WithLimit(n int64) Option {
	return func(uc *Usecase) {
		uc.limit = n
	}
}

func WithDump(dir string) Option {
	return func(uc *Usecase) {
		uc.dumpDir = filepath.Clean(dir)
	}
}
