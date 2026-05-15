package resource

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// RunDelete removes a resource by ID, prompting for confirmation unless force
// is set. The prompt requires the caller to retype the resource's name to
// proceed — matches the safety pattern used for `mass project delete` and
// `mass resource-type delete`.
func RunDelete(ctx context.Context, api API, resourceID string, force bool, in io.Reader) error {
	res, err := api.GetResource(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("error getting resource: %w", err)
	}

	if !force {
		fmt.Printf("WARNING: This will permanently delete resource `%s`.\n", res.Name)
		fmt.Printf("Type `%s` to confirm deletion: ", res.Name)
		reader := bufio.NewReader(in)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != res.Name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deleted, err := api.DeleteResource(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("error deleting resource: %w", err)
	}

	fmt.Printf("Resource %s deleted successfully (ID: %s)\n", deleted.Name, deleted.ID)
	return nil
}
