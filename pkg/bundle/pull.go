package bundle

import (
	"context"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
)

type Puller struct {
	Target oras.Target
	Repo   oras.Target
}

func (p *Puller) PullBundle(ctx context.Context, tag string) (v1.Descriptor, error) {
	return oras.Copy(ctx, p.Repo, tag, p.Target, tag, oras.DefaultCopyOptions)
}
