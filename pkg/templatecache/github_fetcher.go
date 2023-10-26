package templatecache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Function for fetching templates from multiple Github repositories.
func GithubTemplatesFetcher(writePath string) error {
	cloneErrors := []CloneError{}

	for _, repoName := range massdriverApplicationTemplatesRepositories {
		cloneErr := doClone(repoName, writePath)
		// If one repository fails, we want to get the rest so stash the error and handle later.
		if cloneErr != nil {
			errorStruct := CloneError{Repository: repoName, Error: cloneErr.Error()}
			cloneErrors = append(cloneErrors, errorStruct)
		}
	}

	// Given that the Github fetcher writes on each iteration the operation will somewhat succeed. When we return the errors it will document
	// what parts did not.
	if len(cloneErrors) > 0 {
		return concatenateCloneErrors(cloneErrors)
	}

	return nil
}

func concatenateCloneErrors(cloneErrors []CloneError) error {
	concatenatedErrors := errors.New("")
	for _, cloneErr := range cloneErrors {
		concatenatedErrors = fmt.Errorf("%wError fetching repository: %s. Message: %s; ", concatenatedErrors, cloneErr.Repository, cloneErr.Error)
	}

	return concatenatedErrors
}

func doClone(repoName, writePath string) error {
	clonePath := clonePath(repoName, writePath)

	repo, err := git.PlainOpen(clonePath)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryNotExists) {
			return err
		}

		// The error is ErrRepositoryNotExists so we need to clone the repo
		slog.Info("Downloading templates from repo")
		_, cloneErr := git.PlainClone(clonePath, false, &git.CloneOptions{
			URL:   repoName,
			Depth: 1,
		})
		return cloneErr
	}

	latestUpstream, err := getLatestUpstreamCommit(repoName)
	if err != nil {
		return err
	}

	commits, err := repo.CommitObjects()
	if err != nil {
		return err
	}
	latest, err := commits.Next()
	if err != nil {
		return err
	}

	if latestUpstream == latest.Hash.String() {
		slog.Info("Templates are current, skipping download")
		return nil
	}

	// Pull the latest changes from the origin remote and merge into the current branch
	slog.Info("Pulling latest changes from repo")

	if err = os.RemoveAll(clonePath); err != nil {
		return err
	}

	_, err = git.PlainClone(clonePath, false, &git.CloneOptions{
		URL:   repoName,
		Depth: 1,
	})

	return err
}

func clonePath(repoName, writePath string) string {
	pathSuffix := strings.Replace(repoName, "https://github.com", "", 1)
	return fmt.Sprintf("%s%s", writePath, pathSuffix)
}

// Comit represents a commit from the github API, however, we only need the sha at this point
type Commit struct {
	SHA string `json:"sha,omitempty"`
}

func getLatestUpstreamCommit(repo string) (string, error) {
	ghAPI := strings.Replace(repo, "https://github.com", "https://api.github.com/repos", 1)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ghAPI+"/commits?per_page=1", nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var commits []Commit

	err = json.Unmarshal(out, &commits)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return "", nil
	}

	return commits[0].SHA, nil
}
