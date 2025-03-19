package usecase

import (
	"context"
	"github.com/ramk42/omi-backend-assignment/internal/account"
)

type Account struct{}

func NewAccount() *Account {
	return &Account{}
}

func (a *Account) Patch(ctx context.Context, account account.Account) error {

	return nil
}
