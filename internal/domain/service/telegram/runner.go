package telegram

import "context"

type Runner interface {
	Start(ctx context.Context)
}
