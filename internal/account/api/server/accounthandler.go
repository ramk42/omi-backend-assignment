package server

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/ramk42/omi-backend-assignment/internal/account"
	"github.com/ramk42/omi-backend-assignment/internal/account/usecase"
	"net/http"
)

type AccountHandler struct {
	accountUsecase usecase.AccountPort
}

type AccountPatchReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *AccountHandler) Patch(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "resourceID")
	if accountID == "" {
		http.Error(w, "accountID is required", http.StatusBadRequest)
		return
	}

	var req AccountPatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.accountUsecase.Patch(r.Context(), account.Account{
		ID:    accountID,
		Name:  req.Name,
		Email: req.Email,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	return
}
