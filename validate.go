package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/storage/docker/volumeplugin"
	"net/http"
)

var (
	validatePort = ":1000"
	checkOnInit  = true
	driver       *volumeplugin.RancherStorageDriver
)

func startValidation(d *volumeplugin.RancherStorageDriver) {
	driver = d
	http.HandleFunc("/validate", validate)
	log.Infof("Listening for validation on 0.0.0.0%v/validate", validatePort)
	log.Fatal(http.ListenAndServe(validatePort, nil))
}

func validate(w http.ResponseWriter, r *http.Request) {
	if checkOnInit {
		checkOnInit = false
		http.Error(w, "First validation failed as expected", http.StatusInternalServerError)
		return
	}
	err := driver.Validate()
	if err != nil {
		log.Errorf("Validation failed, Rancher cannot talk to the storage: %v", err)
		http.Error(w, "Failed", http.StatusInternalServerError)
	}
}
