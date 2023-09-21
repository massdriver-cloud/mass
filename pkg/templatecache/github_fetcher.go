package templatecache

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Function for fetching templates from multiple Github repositories.
func GithubTemplatesFetcher(writePath string) error {
	cloneErrors := []CloneError{}

	for _, repoName := range massdriverApplicationTemplatesRepositories() {
		fmt.Printf("\tCloning %s\n", repoName)
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

	_, cloneErr := git.PlainClone(clonePath, false, &git.CloneOptions{
		URL:   repoName,
		Depth: 1,
	})

	return cloneErr
}

func clonePath(repoName, writePath string) string {
	pathSuffix := strings.Replace(repoName, "https://github.com", "", 1)
	return fmt.Sprintf("%s%s", writePath, pathSuffix)
}
