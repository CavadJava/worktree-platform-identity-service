package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"platform-identity-service/internal/service"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type createProjectRequest struct {
	Name string `json:"name"`
}

type projectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// Create godoc
// @Summary      Register a new project
// @Description  Açıq endpoint — sistem-səviyyəli superadmin anlayışı hələ yoxdur, yeni layihə qurmaq istəyən istənilən kəs çağıra bilər.
// @Tags         projects
// @Accept       json
// @Produce      json
// @Param        request body createProjectRequest true "Project payload"
// @Success      201 {object} projectResponse
// @Failure      400 {object} map[string]string
// @Router       /projects [post]
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p, err := h.svc.Create(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, projectResponse{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat)})
}

// List godoc
// @Summary      List all projects
// @Tags         projects
// @Produce      json
// @Success      200 {array} projectResponse
// @Router       /projects [get]
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}

	response := make([]projectResponse, len(projects))
	for i, p := range projects {
		response[i] = projectResponse{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat)}
	}
	writeJSON(w, http.StatusOK, response)
}

// Get godoc
// @Summary      Get a project by id
// @Tags         projects
// @Produce      json
// @Param        id path string true "Project ID"
// @Success      200 {object} projectResponse
// @Failure      404 {object} map[string]string
// @Router       /projects/{id} [get]
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, projectResponse{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat)})
}

const timeFormat = "2006-01-02T15:04:05Z07:00"
