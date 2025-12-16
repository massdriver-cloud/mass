package definition

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func Delete(ctx context.Context, mdClient *client.Client, definitionName string, force bool) error {
	// Get definition details for confirmation
	ad, getErr := Get(ctx, mdClient, definitionName)
	if getErr != nil {
		return fmt.Errorf("error getting artifact definition: %w", getErr)
	}

	// Prompt for confirmation - requires typing the definition name unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently delete artifact definition `%s`.\n", ad.Name)
		fmt.Printf("Type `%s` to confirm deletion: ", ad.Name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != ad.Name {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deletedDef, deleteErr := api.DeleteArtifactDefinition(ctx, mdClient, definitionName)
	if deleteErr != nil {
		return fmt.Errorf("error deleting artifact definition: %w", deleteErr)
	}

	fmt.Printf("Artifact definition %s deleted successfully!\n", prettylogs.Underline(deletedDef.Name))
	return nil
}
