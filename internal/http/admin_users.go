package http

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/supermeo/admin"
)

type AdminUsersHandler struct {
	svc *admin.AdminUsersService
}

func NewAdminUsersHandler(svc *admin.AdminUsersService) *AdminUsersHandler {
	return &AdminUsersHandler{svc: svc}
}

func (h *AdminUsersHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/admin/users", requireAuth("owner", h.handleListUsers))
	mux.HandleFunc("GET /v1/admin/users/{id}", requireAuth("owner", h.handleGetUser))
	mux.HandleFunc("DELETE /v1/admin/users/{id}", requireAuth("owner", h.handleDeleteUser))
}

func (h *AdminUsersHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}

	users, total, err := h.svc.ListUsers(r.Context(), search, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if users == nil {
		users = make([]store.User, 0)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":       users,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *AdminUsersHandler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	/* The UserStore interface doesn't have GetUserByID as part of public API,
	   but we keep the route for future expansion. For now, redirect to list. */
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (h *AdminUsersHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
		return
	}

	if err := h.svc.DeleteUser(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
