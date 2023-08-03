#!/bin/bash
# Download the latest release and move to path in a single command.
# example: ./mass-install.sh linux-amd64, ./mass-install.sh darwin-arm64, ./mass-install.sh darwin-amd64
LATEST_RELEASE_URL=$(curl -s https://api.github.com/repos/massdriver-cloud/mass/releases/latest | grep -E "$1.tar.gz\"" | cut -d '"' -f 4 | tail -1)
echo "Latest release = $LATEST_RELEASE_URL"
DOWNLOADED_ARTIFACT=$(echo $LATEST_RELEASE_URL | grep -oP mass-.*-$1.tar.gz)
echo "Artifact was = $DOWNLOADED_ARTIFACT"
wget $LATEST_RELEASE_URL
tar -xzvf $DOWNLOADED_ARTIFACT
sudo mv mass /usr/local/bin/
rm $DOWNLOADED_ARTIFACT
