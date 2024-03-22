package bundle_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
)

var params = map[string]any{
	"params": map[string]interface{}{
		"logLevel":     "ERROR",
		"databaseName": "production",
	},
	"connections": map[string]interface{}{
		"postgres": map[string]interface{}{
			"data": map[string]interface{}{
				"authentication": map[string]interface{}{
					"username": "test",
					"password": "root",
					"hostname": "admin.com",
					"port":     1234,
				},
			},
		},
	},
}

var singleQuery = map[string]string{
	"DATABASE_URL": "@text \"postgres://\" + .connections.postgres.data.authentication.username + \":\" + .connections.postgres.data.authentication.password + \"@\" + .connections.postgres.data.authentication.hostname + \":\" + (.connections.postgres.data.authentication.port|tostring) + \"/\" + .params.databaseName",
}

var multiQuery = map[string]string{
	"DATABASE_URL": "@text \"postgres://\" + .connections.postgres.data.authentication.username + \":\" + .connections.postgres.data.authentication.password + \"@\" + .connections.postgres.data.authentication.hostname + \":\" + (.connections.postgres.data.authentication.port|tostring) + \"/\" + .params.databaseName",
	"LOG_LEVEL":    ".params.logLevel",
}

func TestStringConcatentation(t *testing.T) {
	got := bundle.ParseEnvironmentVariables(params, singleQuery)

	want := "postgres://test:root@admin.com:1234/production"

	if got["DATABASE_URL"].Value != want {
		t.Errorf("Wanted %s but got %s", want, got["DATABASE_URL"].Value)
	}
}

func TestInvalidStringConcatentation(t *testing.T) {
	errorQuery := map[string]string{
		"DATABASE_URL": "@text \"postgres://\" + .connections.postgres.data.authentication.username + \":\" + .connections.postgres.data.authentication.password + \"@\" + .connections.postgres.data.authentication.hostname + \":\" + .connections.postgres.data.authentication.port|tostring + \"/\" + .params.databaseName",
	}

	got := bundle.ParseEnvironmentVariables(params, errorQuery)

	errorMessage := got["DATABASE_URL"].Error

	fmt.Println(errorMessage)

	if errorMessage != "cannot add: string (\"postgres://test:root@adm ...\") and number (1234)" {
		t.Errorf("Wanted %s but got %s", "cannot add: string (\"postgres://test:root@adm ...\") and number (1234)", errorMessage)
	}
}

func TestMultipleValues(t *testing.T) {
	got := bundle.ParseEnvironmentVariables(params, multiQuery)

	asJSON, err := json.Marshal(got)

	if err != nil {
		t.Fatalf("Failed to unmarshal result")
	}

	want := "{\"DATABASE_URL\":{\"error\":\"\",\"value\":\"postgres://test:root@admin.com:1234/production\"},\"LOG_LEVEL\":{\"error\":\"\",\"value\":\"ERROR\"}}"

	if string(asJSON) != want {
		t.Errorf("Wanted %s but got %s", want, asJSON)
	}
}
