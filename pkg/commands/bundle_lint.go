package commands

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

func LintBundle(b *bundle.Bundle) error {
	fmt.Println("Checking massdriver.yaml for errors...")

	err := b.LintSchema()
	if err != nil {
		return err
	}

	err = b.LintParamsConnectionsNameCollision()
	if err != nil {
		return err
	}

	_, err = b.LintEnvs()
	if err != nil {
		return err
	}

	fmt.Println("Linting complete, massdriver.yaml is valid!")

	return nil
}
