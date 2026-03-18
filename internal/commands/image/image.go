package image

// PushImageInput holds all parameters needed to build and push a container image.
type PushImageInput struct {
	ImageName          string
	Location           string
	OrganizationID     string
	Tags               []string
	ArtifactID         string
	Dockerfile         string
	DockerBuildContext string
	TargetPlatform     string
	CacheFrom          string
	SkipBuild          bool
}

// ErrorLine represents an error line returned in a Docker JSON stream response.
type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

// ErrorDetail contains the human-readable message for a Docker error line.
type ErrorDetail struct {
	Message string `json:"message"`
}
