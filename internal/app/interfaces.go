package app

import "context"

type Server interface {
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
