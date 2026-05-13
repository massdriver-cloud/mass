package resourcetype

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// Delete removes a resource type by name, prompting for confirmation unless force is set.
func Delete(ctx context.Context, mdClient *massdriver.Client, name string, force bool) error {
	// Get resource type details for confirmation
	rt, getErr := Get(ctx, mdClient, name)
	if getErr != nil {
		return fmt.Errorf("error getting resource type: %w", getErr)
	}

	// Prompt for confirmation - requires typing the resource type name unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently delete resource type `%s`.\n", rt.Name)
		fmt.Printf("Type `%s` to confirm deletion: ", rt.Name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != rt.Name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deletedRT, deleteErr := api.DeleteResourceType(ctx, mdClient, name)
	if deleteErr != nil {
		return fmt.Errorf("error deleting resource type: %w", deleteErr)
	}

	fmt.Printf("Resource type %s deleted successfully!\n", prettylogs.Underline(deletedRT.Name))
	return nil
}
