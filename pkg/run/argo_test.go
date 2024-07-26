package run_test

import (
	"os"
	"reflect"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/massdriver-cloud/mass/pkg/run"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestTranslateToArgo(t *testing.T) {
	type test struct {
		name         string
		definition   string
		workflowName string
		want         wfv1.Workflow
	}
	tests := []test{
		{
			name:         "simple",
			definition:   `testdata/workflow.yaml`,
			workflowName: "plan",
			want: wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					Entrypoint: "execute",
					Templates: []wfv1.Template{
						{
							Name: "execute",
							Steps: []wfv1.ParallelSteps{
								{
									Steps: []wfv1.WorkflowStep{
										{
											Name:     "rds",
											Template: "rds",
										},
									},
								},
								{
									Steps: []wfv1.WorkflowStep{
										{
											Name:     "infracost",
											Template: "infracost",
											Arguments: wfv1.Arguments{
												Artifacts: wfv1.Artifacts{
													{
														Name: "plan",
														From: "{{steps.rds.outputs.artifacts.plan}}",
													},
												},
											},
										},
									},
								},
								{
									Steps: []wfv1.WorkflowStep{
										{
											Name:     "opa",
											Template: "opa",
											Arguments: wfv1.Arguments{
												Artifacts: wfv1.Artifacts{
													{
														Name: "plan",
														From: "{{steps.rds.outputs.artifacts.plan}}",
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Name: "rds",
							Container: &v1.Container{
								Image: "massdriver/terraform",
								Command: []string{
									"terraform",
									"plan",
									"-out",
									"tfplan.binary",
									"&&",
									"terraform",
									"show",
									"-json",
									"tfplan.binary",
									">",
									"plan.json",
								},
							},
							Outputs: wfv1.Outputs{
								Artifacts: wfv1.Artifacts{
									{
										Name: "plan",
										Path: "./plan.json",
									},
								},
							},
						},
						{
							Name: "infracost",
							Container: &v1.Container{
								Image: "massdriver/infracost",
								Command: []string{
									"infracost",
									"breakdown",
									"--path",
									"plan.json",
								},
							},
							Inputs: wfv1.Inputs{
								Artifacts: wfv1.Artifacts{
									{
										Name: "plan",
										Path: "./plan.json",
									},
								},
							},
						},
						{
							Name: "opa",
							Container: &v1.Container{
								Image: "massdriver/opa",
								Command: []string{
									"opa",
									"exec",
									"--decision",
									"terraform/analysis/authz",
									"--bundle",
									".",
									"plan.json",
								},
							},
							Inputs: wfv1.Inputs{
								Artifacts: wfv1.Artifacts{
									{
										Name: "plan",
										Path: "./plan.json",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytes, err := os.ReadFile(tc.definition)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			definition := run.Definition{}

			err = yaml.Unmarshal(bytes, &definition)
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			got, err := run.TranslateToArgo(*definition.Workflows[tc.workflowName])
			if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if !reflect.DeepEqual(*got, tc.want) {
				gotBytes, _ := yaml.Marshal(*got)
				wantBytes, _ := yaml.Marshal(tc.want)
				t.Errorf("Wanted %s but got %s", string(wantBytes), string(gotBytes))
			}
		})
	}
}
