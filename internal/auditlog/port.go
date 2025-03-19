package auditlog

import "context"

type Repository interface {
	Insert(ctx context.Context, events []*Model) error
}

type Usecase interface {
	Push(ctx context.Context, model *Model) error
	Close()
}
