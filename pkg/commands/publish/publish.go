package publish

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Publisher struct {
	Store oras.Target
	Repo  oras.Target
}

func (p *Publisher) PublishBundle(ctx context.Context, tag string) error {
	_, copyErr := oras.Copy(ctx, p.Store, tag, p.Repo, tag, oras.DefaultCopyOptions)
	return copyErr
}

func getRepo(b *bundle.Bundle, mdClient *client.Client) (oras.Target, error) {
	if mdClient.Auth.Method != config.AuthAPIKey {
		return nil, fmt.Errorf("bundle publish requires API key auth")
	}
	// reg := mdClient.Auth.URL
	// repo, repoErr := remote.NewRepository(filepath.Join(reg, mdClient.Auth.AccountID, b.Name))
	reg := "2d67-47-229-209-228.ngrok-free.app"
	repo, repoErr := remote.NewRepository(filepath.Join(reg, "sandbox", b.Name))
	if repoErr != nil {
		return nil, repoErr
	}

	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(reg, auth.Credential{
			Username: "myuser",
			Password: "mypass",
		}),
	}

	return repo, nil
}
