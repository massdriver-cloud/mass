package image

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
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}
