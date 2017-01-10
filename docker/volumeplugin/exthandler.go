package volumeplugin

import (
	"github.com/docker/go-plugins-helpers/sdk"
	"github.com/docker/go-plugins-helpers/volume"
	"net/http"
)

const (
	attachPath = "/VolumeDriver.Attach"
)

type ExtDriver interface {
	Attach(AttachRequest) volume.Response
}

type AttachRequest struct {
	Name string
	ID   string
}

type attachActionHandler func(AttachRequest) volume.Response

func ExtendHandler(h *volume.Handler, d ExtDriver) {
	handleAttach(h, attachPath, func(req AttachRequest) volume.Response {
		return d.Attach(req)
	})
}

func handleAttach(h *volume.Handler, name string, actionCall attachActionHandler) {
	h.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		var req AttachRequest
		if err := sdk.DecodeRequest(w, r, &req); err != nil {
			return
		}
		res := actionCall(req)
		sdk.EncodeResponse(w, res, res.Err)
	})
}
