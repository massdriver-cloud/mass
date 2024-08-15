package bundle_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

func TestPoop(t *testing.T) {
	bundle.ImportParams("testdata/import")

	t.Fail()
}
