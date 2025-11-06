package files

import (
	"os"
	"reflect"
	"testing"
)

func TestRead_JSON(t *testing.T) {
	var result map[string]interface{}
	err := Read("testdata/test.json", &result)
	if err != nil {
		t.Fatalf("Failed to read test.json: %v", err)
	}

	expected := map[string]interface{}{
		"name":    "test",
		"value":   float64(42), // JSON unmarshals numbers as float64
		"enabled": true,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("JSON result mismatch: got %v, want %v", result, expected)
	}
}

func TestRead_TOML(t *testing.T) {
	var result map[string]interface{}
	err := Read("testdata/test.toml", &result)
	if err != nil {
		t.Fatalf("Failed to read test.toml: %v", err)
	}

	expected := map[string]interface{}{
		"name":    "test",
		"value":   int64(42),
		"enabled": true,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("TOML result mismatch: got %v, want %v", result, expected)
	}
}

func TestRead_YAML(t *testing.T) {
	var result map[string]interface{}
	err := Read("testdata/test.yaml", &result)
	if err != nil {
		t.Fatalf("Failed to read test.yaml: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name to be 'test', got %v", result["name"])
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled to be true, got %v", result["enabled"])
	}
	// YAML might parse numbers as int64 or int
	value := result["value"]
	if v, ok := value.(int64); ok && v != 42 {
		t.Errorf("Expected value to be 42, got %v", v)
	} else if v, ok := value.(int); ok && v != 42 {
		t.Errorf("Expected value to be 42, got %v", v)
	} else if v, ok := value.(float64); ok && v != 42 {
		t.Errorf("Expected value to be 42, got %v", v)
	} else if !ok {
		t.Errorf("Expected value to be a number, got %T: %v", value, value)
	}
}

func TestRead_TFVARS(t *testing.T) {
	var result map[string]interface{}
	err := Read("testdata/test.tfvars", &result)
	if err != nil {
		t.Fatalf("Failed to read test.tfvars: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name to be 'test', got %v", result["name"])
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled to be true, got %v", result["enabled"])
	}
	// TFVARS might parse numbers as int64 or float64 depending on how ctyjson marshals
	value := result["value"]
	if v, ok := value.(int64); ok && v != 42 {
		t.Errorf("Expected value to be 42, got %v", v)
	} else if v, ok := value.(int); ok && v != 42 {
		t.Errorf("Expected value to be 42, got %v", v)
	} else if v, ok := value.(float64); ok && v != 42 {
		t.Errorf("Expected value to be 42, got %v", v)
	} else if !ok {
		t.Errorf("Expected value to be a number, got %T: %v", value, value)
	}
}

func TestRead_ComplexTFVars(t *testing.T) {
	var result map[string]interface{}
	err := Read("testdata/complex.tfvars", &result)
	if err != nil {
		t.Fatalf("Failed to read complex.tfvars: %v", err)
	}

	// Verify that the file was parsed successfully
	// Check for some expected top-level keys
	expectedKeys := []string{"capacity", "global_secondary_indexes", "pitr", "primary_index", "region", "stream", "ttl"}
	for _, key := range expectedKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("Missing expected key: %s", key)
		}
	}

	// Verify some nested structure
	if capacity, ok := result["capacity"].(map[string]interface{}); ok {
		if billingMode, ok := capacity["billing_mode"].(string); !ok || billingMode != "PROVISIONED" {
			t.Errorf("Expected capacity.billing_mode to be 'PROVISIONED', got %v", capacity["billing_mode"])
		}
		if readCap, ok := capacity["read_capacity"].(float64); !ok || readCap != 20 {
			t.Errorf("Expected capacity.read_capacity to be 20, got %v", capacity["read_capacity"])
		}
		if writeCap, ok := capacity["write_capacity"].(float64); !ok || writeCap != 50 {
			t.Errorf("Expected capacity.write_capacity to be 50, got %v", capacity["write_capacity"])
		}
	} else {
		t.Errorf("Expected capacity to be a map, got %T", result["capacity"])
	}

	// Verify region
	if region, ok := result["region"].(string); !ok || region != "us-east-1" {
		t.Errorf("Expected region to be 'us-east-1', got %v", result["region"])
	}

	// Verify pitr
	if pitr, ok := result["pitr"].(map[string]interface{}); ok {
		if enabled, ok := pitr["enabled"].(bool); !ok || !enabled {
			t.Errorf("Expected pitr.enabled to be true, got %v", pitr["enabled"])
		}
	} else {
		t.Errorf("Expected pitr to be a map, got %T", result["pitr"])
	}

	// Verify global_secondary_indexes is an array
	if gsi, ok := result["global_secondary_indexes"].([]interface{}); ok {
		if len(gsi) != 2 {
			t.Errorf("Expected global_secondary_indexes to have 2 items, got %d", len(gsi))
		}
		if len(gsi) > 0 {
			firstIndex := gsi[0].(map[string]interface{})
			if name, ok := firstIndex["name"].(string); !ok || name != "user-audit-index" {
				t.Errorf("Expected first index name to be 'user-audit-index', got %v", firstIndex["name"])
			}
		}
	} else {
		t.Errorf("Expected global_secondary_indexes to be an array, got %T", result["global_secondary_indexes"])
	}

	// Verify primary_index
	if primaryIndex, ok := result["primary_index"].(map[string]interface{}); ok {
		if pType, ok := primaryIndex["type"].(string); !ok || pType != "compound" {
			t.Errorf("Expected primary_index.type to be 'compound', got %v", primaryIndex["type"])
		}
		if partitionKey, ok := primaryIndex["partition_key"].(string); !ok || partitionKey != "entity_id" {
			t.Errorf("Expected primary_index.partition_key to be 'entity_id', got %v", primaryIndex["partition_key"])
		}
	} else {
		t.Errorf("Expected primary_index to be a map, got %T", result["primary_index"])
	}

	// Verify stream
	if stream, ok := result["stream"].(map[string]interface{}); ok {
		if enabled, ok := stream["enabled"].(bool); !ok || !enabled {
			t.Errorf("Expected stream.enabled to be true, got %v", stream["enabled"])
		}
		if viewType, ok := stream["view_type"].(string); !ok || viewType != "NEW_AND_OLD_IMAGES" {
			t.Errorf("Expected stream.view_type to be 'NEW_AND_OLD_IMAGES', got %v", stream["view_type"])
		}
	} else {
		t.Errorf("Expected stream to be a map, got %T", result["stream"])
	}

	// Verify ttl
	if ttl, ok := result["ttl"].(map[string]interface{}); ok {
		if enabled, ok := ttl["enabled"].(bool); !ok || enabled {
			t.Errorf("Expected ttl.enabled to be false, got %v", ttl["enabled"])
		}
	} else {
		t.Errorf("Expected ttl to be a map, got %T", result["ttl"])
	}
}

func TestRead_UnsupportedFileType(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile := t.TempDir() + "/test.txt"
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	err = Read(tmpFile, &result)
	if err == nil {
		t.Fatal("Expected error for unsupported file type, got nil")
	}
	if err.Error() != "unsupported file type: .txt" {
		t.Errorf("Expected error message about unsupported file type, got: %v", err)
	}
}

func TestRead_FileNotFound(t *testing.T) {
	var result map[string]interface{}
	err := Read("testdata/nonexistent.json", &result)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}
