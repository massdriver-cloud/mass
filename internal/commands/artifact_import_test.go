package commands_test

// func TestArtifactImport(t *testing.T) {
// 	responses := []interface{}{
// 		gqlmock.MockQueryResponse("getArtifactDefinitions", []api.ArtifactDefinitionWithSchema{
// 			{
// 				Name: "massdriver/fake-artifact-schema",
// 				Schema: map[string]interface{}{
// 					"$id":     "id",
// 					"$schema": "http://json-schema.org/draft-07/schema",
// 					"type":    "object",
// 					"properties": map[string]interface{}{
// 						"name": map[string]interface{}{
// 							"type": "string",
// 						},
// 					},
// 				},
// 			},
// 		}),
// 		gqlmock.MockMutationResponse("createArtifact", api.Artifact{
// 			ID:   "artifact-id",
// 			Name: "artifact-name",
// 		}),
// 	}

// 	client := gqlmock.NewClientWithJSONResponseArray(responses)

// 	var fs = afero.NewMemMapFs()

// 	file, err := fs.Create("artifact.json")
// 	checkErr(err, t)

// 	file.Write([]byte(`{"name":"fake"}`))

// 	got, err := commands.ArtifactImport(client, "faux-org-id", fs, "artifact-name", "massdriver/fake-artifact-schema", "artifact.json")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	want := "artifact-id"
// 	if got != want {
// 		t.Errorf("got %s , wanted %s", got, want)
// 	}
// }
