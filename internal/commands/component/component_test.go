package component_test

import (
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/component"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

func TestSplitComponentID(t *testing.T) {
	tests := []struct {
		in      string
		wantP   string
		wantC   string
		wantErr bool
	}{
		{in: "ecomm-db", wantP: "ecomm", wantC: "db"},
		{in: "api-database", wantP: "api", wantC: "database"},
		{in: "", wantErr: true},
		{in: "ecomm", wantErr: true},
		{in: "-db", wantErr: true},
		{in: "ecomm-", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			gotP, gotC, err := component.SplitComponentID(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotP != tc.wantP || gotC != tc.wantC {
				t.Errorf("got (%q, %q), want (%q, %q)", gotP, gotC, tc.wantP, tc.wantC)
			}
		})
	}
}

func TestParseComponentField(t *testing.T) {
	tests := []struct {
		in       string
		wantComp string
		wantF    string
		wantErr  bool
	}{
		{in: "ecomm-db.authentication", wantComp: "ecomm-db", wantF: "authentication"},
		{in: "x.y", wantComp: "x", wantF: "y"},
		{in: "", wantErr: true},
		{in: "ecomm-db", wantErr: true},
		{in: ".authentication", wantErr: true},
		{in: "ecomm-db.", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			gotC, gotF, err := component.ParseComponentField(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotC != tc.wantComp || gotF != tc.wantF {
				t.Errorf("got (%q, %q), want (%q, %q)", gotC, gotF, tc.wantComp, tc.wantF)
			}
		})
	}
}

func TestFindLink(t *testing.T) {
	from := &types.Component{ID: "ecomm-db"}
	to := &types.Component{ID: "ecomm-app"}
	links := []types.Link{
		{ID: "wrong-link", FromField: "other", ToField: "database", FromComponent: from, ToComponent: to},
		{ID: "target-link", FromField: "authentication", ToField: "database", FromComponent: from, ToComponent: to},
	}

	got, err := component.FindLink(links, "ecomm-db", "authentication", "ecomm-app", "database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "target-link" {
		t.Errorf("got %q, want target-link", got.ID)
	}

	if _, err := component.FindLink(links, "ecomm-db", "missing", "ecomm-app", "database"); err == nil {
		t.Error("expected error when no link matches, got none")
	}
}
