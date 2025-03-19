package usecase

import (
	"context"
	"github.com/ramk42/omi-backend-assignment/internal/account"
)

type AccountPort interface {
	Patch(ctx context.Context, account account.Account) error
}
