package publish

var mimeTypesFromExt = map[string]string{
	// Text formats
	".txt": "text/plain",
	".md":  "text/markdown",
	".mdx": "text/markdown",
	".csv": "text/csv",
	".log": "text/plain",
	// Configuration / serialization
	".json": "application/json",
	".yaml": "application/yaml",
	".yml":  "application/yaml",
	".toml": "application/toml",
	".ini":  "text/plain", // technically ambiguous
	// HTML, XML
	".html": "text/html",
	".xml":  "application/xml",
	// Source code
	".go":   "text/x-go",
	".py":   "text/x-python",
	".js":   "application/javascript",
	".ts":   "application/typescript",
	".java": "text/x-java-source",
	".rb":   "text/x-ruby",
	".sh":   "application/x-sh",
	".bash": "application/x-sh",
	".c":    "text/x-c",
	".cpp":  "text/x-c++",
	".cs":   "text/x-csharp",
	".php":  "application/x-httpd-php",
	// Infrastructure as code / DevOps
	".tf":         "application/hcl",
	".tfvars":     "application/hcl",
	".hcl":        "application/hcl",
	".rego":       "text/plain", // Open Policy Agent
	".dockerfile": "text/x-dockerfile",
	// Shell scripts / dotfiles
	".env":           "text/plain",
	".gitignore":     "text/plain",
	".gitattributes": "text/plain",
	".bashrc":        "text/x-shellscript",
	// Archives
	".zip":    "application/x-zip-compressed",
	".tar":    "application/x-tar",
	".gz":     "application/x-gzip",
	".tgz":    "application/x-gzip",
	".tar.gz": "application/x-gzip",
	// Binary
	".exe":  "application/vnd.microsoft.portable-executable",
	".dll":  "application/vnd.microsoft.portable-executable",
	".wasm": "application/wasm",
	// Images (commonly used in docs/pipelines)
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	// Certificates / keys
	".pem": "application/x-pem-file",
	".crt": "application/x-x509-ca-cert",
	".key": "application/x-pem-file",
}

func getMimeTypeFromExtension(ext string) string {
	if mimeType, exists := mimeTypesFromExt[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}
