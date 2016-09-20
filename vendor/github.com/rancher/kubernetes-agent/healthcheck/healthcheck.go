package healthcheck

import (
	"fmt"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

func healthcheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func StartHealthCheck(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("Invalid health check port number: %v", port)
	}
	http.HandleFunc("/healthcheck", healthcheck)
	p := ":" + strconv.Itoa(port)
	log.Infof("Listening for health checks on 0.0.0.0%v/healthcheck", p)
	err := http.ListenAndServe(p, nil)
	return err
}
