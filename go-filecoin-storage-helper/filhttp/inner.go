package filhttp

import (
	"context"
	"io"
)

type Storage interface {
	Cat(ctx context.Context, cid string) (io.Reader, error)

	Import(ctx context.Context, fr io.Reader) (string, error)

	//TODO: Storage order cannot be created at this time, pending verification
	ProposeStorageDeal(ctx context.Context, miner string, cid string, askId string, time int64) (*DealInfo, error)

	//TODO: pending verification
	QueryStorageDeal(ctx context.Context, dealID string) (*DealInfo, error)
}
