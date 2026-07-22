package repository_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/repository"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
)

// TestArtifactTypeRoundTrip verifies that every user-facing alias resolves to
// an enum and renders back to the same friendly label, for both supported
// artifact types.
func TestArtifactTypeRoundTrip(t *testing.T) {
	cases := []struct {
		alias string
		enum  ocirepos.ArtifactType
	}{
		{"bundle", ocirepos.ArtifactTypeBundle},
		{"resource-type", ocirepos.ArtifactTypeResourceType},
	}

	for _, tc := range cases {
		t.Run(tc.alias, func(t *testing.T) {
			at, err := repository.ResolveArtifactType(tc.alias)
			if err != nil {
				t.Fatalf("ResolveArtifactType(%q) returned error: %v", tc.alias, err)
			}
			if at != tc.enum {
				t.Errorf("ResolveArtifactType(%q) = %q, want %q", tc.alias, at, tc.enum)
			}
			if label := repository.ArtifactTypeLabel(at); label != tc.alias {
				t.Errorf("ArtifactTypeLabel(%q) = %q, want %q", at, label, tc.alias)
			}
		})
	}
}

func TestResolveArtifactTypeCaseInsensitive(t *testing.T) {
	at, err := repository.ResolveArtifactType("Resource-Type")
	if err != nil {
		t.Fatalf("ResolveArtifactType returned error: %v", err)
	}
	if at != ocirepos.ArtifactTypeResourceType {
		t.Errorf("ResolveArtifactType(%q) = %q, want %q", "Resource-Type", at, ocirepos.ArtifactTypeResourceType)
	}
}

func TestResolveArtifactTypeUnknown(t *testing.T) {
	_, err := repository.ResolveArtifactType("provisioner")
	if err == nil {
		t.Fatal("expected error for unknown artifact type, got nil")
	}
	// The error should enumerate the valid types in sorted order.
	want := "unknown artifact type \"provisioner\" (valid: bundle, resource-type)"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// TestArtifactTypeLabelFallback ensures an unmapped enum value renders as its
// raw string rather than an empty label.
func TestArtifactTypeLabelFallback(t *testing.T) {
	if got := repository.ArtifactTypeLabel(ocirepos.ArtifactType("SOMETHING_NEW")); got != "SOMETHING_NEW" {
		t.Errorf("ArtifactTypeLabel fallback = %q, want %q", got, "SOMETHING_NEW")
	}
}
