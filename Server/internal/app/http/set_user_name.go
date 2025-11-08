package http

import (
	"context"
	"io"
	"net/http"

	"github.com/inzarubin80/Server/internal/app/defenitions"
	"github.com/inzarubin80/Server/internal/app/uhttp"
	"github.com/inzarubin80/Server/internal/model"
)

type (
	serviceSetUserName interface {
		SetUserName(ctx context.Context, userID model.UserID, name string) error
	}
	SetUserNameHandler struct {
		name    string
		service serviceSetUserName
	}
)

func NewSetUserNameHandler(service serviceSetUserName, name string) *SetUserNameHandler {
	return &SetUserNameHandler{
		name:    name,
		service: service,
	}
}

func (h *SetUserNameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	userID, ok := ctx.Value(defenitions.UserID).(model.UserID)
	if !ok {
		uhttp.SendErrorResponse(w, http.StatusInternalServerError, "not user ID")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		uhttp.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = h.service.SetUserName(ctx, model.UserID(userID), string(body))
	if err != nil {
		uhttp.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
	}

	uhttp.SendSuccessfulResponse(w, []byte("{}"))

}
