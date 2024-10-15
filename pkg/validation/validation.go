package validation

import (
	"github.com/docker/docker/errdefs"
	"strings"

	"github.com/containrrr/watchtower/pkg/container"
	"github.com/containrrr/watchtower/pkg/types"
	log "github.com/sirupsen/logrus"

	"golang.org/x/net/context"
)

func ValidateParams(client container.Client, params types.UpdateParams) error {
	containers, err := client.ListContainers(params.Filter)
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		return types.NewValidationError("no containers found")
	}
	if params.NewImageName != "" {
		for _, c := range containers {
			if !c.IsPMM() {
				return types.NewValidationError("container is not a PMM server")
			}

			c.SetNewImageName(params.NewImageName)
			log.Tracef("PMM container %s new image name is %s", c.Name(), params.NewImageName)
		}
	}

	if !isImageAllowed(params.AllowedImageRepos, params.NewImageName) {
		return types.NewValidationError("image not allowed")
	}

	hasNew, _, err := client.HasNewImage(context.TODO(), containers[0])
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return err
		}
		log.Debugf("Image not found locally, checking remotely: %s", err)
	}
	// if new image is available locally, we don't need to check remotely.
	if hasNew {
		return nil
	}
	pullNeeded, err := client.PullNeeded(context.TODO(), containers[0])
	if err != nil {
		return err
	}
	// if pull is needed, we don't need to check digest
	if !pullNeeded {
		return types.NewValidationError("no new image available")
	}
	return nil
}

func isImageAllowed(repos []string, newImageName string) bool {
	newImageName = strings.TrimPrefix(newImageName, "docker.io/")
	if newImageName == "" || len(repos) == 0 {
		return true
	}
	for _, repo := range repos {
		repo = strings.TrimPrefix(repo, "docker.io/")
		if strings.HasPrefix(newImageName, repo) {
			return true
		}
	}
	return false
}
