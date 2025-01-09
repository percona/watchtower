package update

import (
	"errors"
	"github.com/containrrr/watchtower/pkg/types"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	lock chan bool
)

// New is a factory function creating a new  Handler instance
func New(updateFn func(images []string, hostname string, newImageName string, stopWatchtower bool) error, updateLock chan bool) *Handler {
	if updateLock != nil {
		lock = updateLock
	} else {
		lock = make(chan bool, 1)
		lock <- true
	}

	return &Handler{
		fn:   updateFn,
		Path: "/v1/update",
	}
}

// Handler is an API handler used for triggering container update scans
type Handler struct {
	fn   func(images []string, hostname string, newImageName string, stopWatchtower bool) error
	Path string
}

// Handle is the actual http.Handle function doing all the heavy lifting
func (handle *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	log.Info("Updates triggered by HTTP API request.")
	log.Debugf("Request received: %s", r.URL.String())

	_, err := io.Copy(os.Stdout, r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	var images []string
	imageQueries, found := r.URL.Query()["image"]
	if found {
		log.Debugf("Image parameter found: %s", imageQueries)
		for _, image := range imageQueries {
			images = append(images, strings.Split(image, ",")...)
		}
	} else {
		images = nil
	}

	// Retrieve the hostname parameter from the URL query parameters
	var hostname string
	hostnameParams, found := r.URL.Query()["hostname"]
	if found {
		log.Debugf("Hostname parameter found: %s", hostnameParams[0])
		hostname = hostnameParams[0]
	}

	var newImageName string
	newImageNameParams, found := r.URL.Query()["newImageName"]
	if found {
		log.Debugf("New image name parameter found: %s", newImageNameParams[0])
		newImageName = newImageNameParams[0]
	}

	var stopWatchtower bool
	stopWatchtowerParams, found := r.URL.Query()["stopWatchtower"]
	if found {
		log.Debugf("Stop watchtower parameter found: %s", stopWatchtowerParams[0])
		stopWatchtower, err = strconv.ParseBool(stopWatchtowerParams[0])
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid stopWatchtower parameter"))
			return
		}
	}

	if len(images) > 0 {
		chanValue := <-lock
		defer func() { lock <- chanValue }()
		err := handle.fn(images, hostname, newImageName, stopWatchtower)
		handleError(w, err)
	} else {
		select {
		case chanValue := <-lock:
			defer func() { lock <- chanValue }()
			err := handle.fn(images, hostname, newImageName, stopWatchtower)
			handleError(w, err)
		default:
			log.Debug("Skipped. Another update already running.")
		}
	}

}

func handleError(w http.ResponseWriter, err error) {
	if err != nil {
		if errors.Is(err, &types.ValidationError{}) {
			log.Warning(err)
			w.WriteHeader(http.StatusPreconditionFailed)
			w.Write([]byte(err.Error()))
		} else {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
