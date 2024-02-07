# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.217.4/containers/go/.devcontainer/base.Dockerfile

FROM mcr.microsoft.com/devcontainers/go:1.22-bullseye

# [Choice] Node.js version: none, lts/*, 16, 14, 12, 10
ARG NODE_VERSION="none"
RUN if [ "${NODE_VERSION}" != "none" ]; then su vscode -c "umask 0002 && . /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# install golangci-lint and goimportsk
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.1 && \
  go install golang.org/x/tools/cmd/goimports@latest

# Add write permission for /go/pkg
RUN chmod -R a+w /go/pkg

ENV DEBIAN_FRONTEND=noninteractive
# install python / pre-commit
RUN sudo apt update && \
  sudo apt install --no-install-recommends -y gcc musl-dev python3-dev python3-venv python3-pip && \
  pip install setuptools wheel ruamel.yaml.clib==0.2.6 pre-commit


# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>

# [Optional] Uncomment the next lines to use go get to install anything else you need
# USER vscode
# RUN go get -x <your-dependency-or-tool>

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1
