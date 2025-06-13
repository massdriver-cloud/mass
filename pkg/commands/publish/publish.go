package publish

import (
	"context"

	oras "oras.land/oras-go/v2"
)

type Publisher struct {
	Store oras.Target
	Repo  oras.Target
}

func (p *Publisher) PublishBundle(ctx context.Context, tag string) error {
	_, copyErr := oras.Copy(ctx, p.Store, tag, p.Repo, tag, oras.DefaultCopyOptions)
	return copyErr
}
