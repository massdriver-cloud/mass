package run

type Definition struct {
	Name      string               `yaml:"name"`
	Workflows map[string]*Workflow `yaml:"workflows"`
}

type Workflow struct {
	Steps []WorkflowStep `yaml:"steps"`
}

type StepInputs struct {
	From string `yaml:"from"`
	Path string `yaml:"path"`
}

type StepOutputs struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type WorkflowStep struct {
	Name    string         `yaml:"name"`
	Image   string         `yaml:"image"`
	Path    string         `yaml:"path"`
	With    map[string]any `yaml:"with"`
	Command string         `yaml:"run"`
	Inputs  []StepInputs   `yaml:"inputs"`
	Outputs []StepOutputs  `yaml:"outputs"`
}
