package update

import (
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	lock chan bool
)

// New is a factory function creating a new  Handler instance
func New(updateFn func(images []string, hostname string, newImageName string) error, updateLock chan bool) *Handler {
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
	fn   func(images []string, hostname string, newImageName string) error
	Path string
}

// Handle is the actual http.Handle function doing all the heavy lifting
func (handle *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	log.Info("Updates triggered by HTTP API request.")

	_, err := io.Copy(os.Stdout, r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	var images []string
	imageQueries, found := r.URL.Query()["image"]
	if found {
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
		hostname = hostnameParams[0]
	}

	var newImageName string
	newImageNameParams, found := r.URL.Query()["newImageName"]
	if found {
		newImageName = newImageNameParams[0]
	}

	if len(images) > 0 {
		chanValue := <-lock
		defer func() { lock <- chanValue }()
		err := handle.fn(images, hostname, newImageName)
		if err != nil {
			log.Error(err)
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		select {
		case chanValue := <-lock:
			defer func() { lock <- chanValue }()
			err := handle.fn(images, hostname, newImageName)
			if err != nil {
				log.Error(err)
				w.Write([]byte(err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
			}
		default:
			log.Debug("Skipped. Another update already running.")
		}
	}

}
