package run

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// var helloWorldWorkflow = wfv1.Workflow{
// 	ObjectMeta: metav1.ObjectMeta{
// 		GenerateName: "hello-world-",
// 	},
// 	Spec: wfv1.WorkflowSpec{
// 		Entrypoint: "whalesay",
// 		Templates: []wfv1.Template{
// 			{
// 				Name: "whalesay",
// 				Container: &corev1.Container{
// 					Image:   "docker/whalesay:latest",
// 					Command: []string{"cowsay", "hello world"},
// 				},
// 			},
// 		},
// 	},
// }

func TranslateToArgo(bundleWorkflow Workflow) (*wfv1.Workflow, error) {
	argoWorkflow := new(wfv1.Workflow)

	argoWorkflow.Spec = wfv1.WorkflowSpec{
		Entrypoint: "execute",
	}

	executeTemplate := wfv1.Template{
		Name:  "execute",
		Steps: []wfv1.ParallelSteps{},
	}

	templates := []wfv1.Template{}

	for _, step := range bundleWorkflow.Steps {
		argoWorkflowStep := wfv1.WorkflowStep{
			Name:     step.Name,
			Template: step.Name,
		}

		template := wfv1.Template{
			Name: step.Name,
			Container: &corev1.Container{
				Image:   step.Image,
				Command: strings.Split(step.Command, " "),
			},
		}

		for _, input := range step.Inputs {
			split := strings.Split(input.From, ".")
			fromStep := split[0]
			fromName := split[1]

			template.Inputs.Artifacts = append(template.Inputs.Artifacts, wfv1.Artifact{
				Name: fromName,
				Path: input.Path,
			})

			argoWorkflowStep.Arguments.Artifacts = append(argoWorkflowStep.Arguments.Artifacts, wfv1.Artifact{
				Name: fromName,
				From: fmt.Sprintf("{{steps.%s.outputs.artifacts.%s}}", fromStep, fromName),
			})
		}

		for _, output := range step.Outputs {
			template.Outputs.Artifacts = append(template.Outputs.Artifacts, wfv1.Artifact{
				Name: output.Name,
				Path: output.Path,
			})
		}

		executeTemplate.Steps = append(executeTemplate.Steps, wfv1.ParallelSteps{Steps: []wfv1.WorkflowStep{argoWorkflowStep}})
		templates = append(templates, template)
	}

	argoWorkflow.Spec.Templates = append(argoWorkflow.Spec.Templates, executeTemplate)
	argoWorkflow.Spec.Templates = append(argoWorkflow.Spec.Templates, templates...)

	return argoWorkflow, nil
}

// func foo() {
// 	// get current user to determine home directory
// 	usr, err := user.Current()
// 	checkErr(err)

// 	// get kubeconfig file location
// 	kubeconfig := flag.String("kubeconfig", filepath.Join(usr.HomeDir, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
// 	flag.Parse()

// 	// use the current context in kubeconfig
// 	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
// 	checkErr(err)
// 	namespace := "default"

// 	// create the workflow client
// 	wfClient := wfclientset.NewForConfigOrDie(config).ArgoprojV1alpha1().Workflows(namespace)

// 	// submit the hello world workflow
// 	ctx := context.Background()
// 	createdWf, err := wfClient.Create(ctx, &helloWorldWorkflow, metav1.CreateOptions{})
// 	checkErr(err)
// 	fmt.Printf("Workflow %s submitted\n", createdWf.Name)

// 	// wait for the workflow to complete
// 	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", createdWf.Name))
// 	watchIf, err := wfClient.Watch(ctx, metav1.ListOptions{FieldSelector: fieldSelector.String(), TimeoutSeconds: pointer.Int64(180)})
// 	errors.CheckError(err)
// 	defer watchIf.Stop()
// 	for next := range watchIf.ResultChan() {
// 		wf, ok := next.Object.(*wfv1.Workflow)
// 		if !ok {
// 			continue
// 		}
// 		if !wf.Status.FinishedAt.IsZero() {
// 			fmt.Printf("Workflow %s %s at %v. Message: %s.\n", wf.Name, wf.Status.Phase, wf.Status.FinishedAt, wf.Status.Message)
// 			break
// 		}
// 	}
// }

// func checkErr(err error) {
// 	if err != nil {
// 		panic(err.Error())
// 	}
// }
