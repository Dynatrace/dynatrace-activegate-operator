package version

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
)

const VersionKey = "version"

type DockerLabelsChecker struct {
	image        string
	labels       map[string]string
	dockerConfig *parser.DockerConfig
}

func NewDockerLabelsChecker(image string, labels map[string]string, dockerConfig *parser.DockerConfig) *DockerLabelsChecker {
	return &DockerLabelsChecker{
		image:        image,
		labels:       labels,
		dockerConfig: dockerConfig,
	}
}

func (dockerLabelsChecker *DockerLabelsChecker) IsLatest() (bool, error) {
	versionLabel, hasVersionLabel := dockerLabelsChecker.labels[VersionKey]
	if !hasVersionLabel {
		return false, fmt.Errorf("key '%s' not found in given labels", VersionKey)
	}

	remoteVersionLabel, err := GetVersionLabel(dockerLabelsChecker.image, dockerLabelsChecker.dockerConfig)
	if err != nil {
		return false, err
	}

	localVersion, err := ExtractVersion(versionLabel)
	if err != nil {
		return false, err
	}

	remoteVersion, err := ExtractVersion(remoteVersionLabel)
	if err != nil {
		return false, err
	}

	// Return true if local version is equal or greater to the remote version
	return CompareVersionInfo(localVersion, remoteVersion) >= 0, nil
}
