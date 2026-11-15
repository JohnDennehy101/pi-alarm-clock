package alarm

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, alarm *Alarm) error
	Get(ctx context.Context, id string) (*Alarm, error)
	List(ctx context.Context) ([]Alarm, error)
	Update(ctx context.Context, alarm *Alarm) error
	Delete(ctx context.Context, id string) error
}
