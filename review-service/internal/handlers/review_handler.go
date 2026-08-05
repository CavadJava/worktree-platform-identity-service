package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"review-service/internal/middleware"
	"review-service/internal/service"
)

// allowedMedia maps an accepted multipart Content-Type to (media_type, file
// extension). Anything outside this whitelist is rejected — this is a demo
// review system, not a general file host.
var allowedMedia = map[string]struct {
	kind string
	ext  string
}{
	"image/jpeg":      {"image", ".jpg"},
	"image/png":       {"image", ".png"},
	"image/webp":      {"image", ".webp"},
	"image/heic":      {"image", ".heic"},
	"image/heif":      {"image", ".heif"},
	"video/mp4":       {"video", ".mp4"},
	"video/quicktime": {"video", ".mov"},
}

type ReviewHandler struct {
	svc       *service.ReviewService
	uploadDir string
	maxBytes  int64
}

func NewReviewHandler(svc *service.ReviewService, uploadDir string, maxBytes int64) *ReviewHandler {
	return &ReviewHandler{svc: svc, uploadDir: uploadDir, maxBytes: maxBytes}
}

type reviewRequest struct {
	Rating int    `json:"rating"`
	Text   string `json:"text"`
}

// Create godoc
// @Summary      Write a review for a product
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        product_id path string true "Product ID"
// @Param        request body reviewRequest true "Review payload"
// @Success      201 {object} models.Review
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /products/{product_id}/reviews [post]
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "product_id")

	var req reviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	review, err := h.svc.Create(r.Context(), userID, productID, req.Rating, req.Text)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

// List godoc
// @Summary      List a product's reviews (newest first)
// @Tags         reviews
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {array} models.Review
// @Router       /products/{product_id}/reviews [get]
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "product_id")
	reviews, err := h.svc.ListByProduct(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list reviews")
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

// Delete godoc
// @Summary      Delete your own review
// @Tags         reviews
// @Security     BearerAuth
// @Param        id path string true "Review ID"
// @Success      204 "No Content"
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /reviews/{id} [delete]
func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		writeReviewError(w, err)
		return
	}

	// Best-effort: the review's media lives under one directory keyed by
	// its ID, so cleanup is a single RemoveAll. A failure here just leaves
	// an orphaned file on disk — never blocks the (already-successful) delete.
	_ = os.RemoveAll(filepath.Join(h.uploadDir, "reviews", id))

	w.WriteHeader(http.StatusNoContent)
}

// AddMedia godoc
// @Summary      Attach a photo or short clip to your review
// @Description  multipart/form-data with a single "file" field. Images: jpeg/png/webp/heic. Video: mp4/mov. Max 25MB.
// @Tags         reviews
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Review ID"
// @Param        file formData file true "Photo or video file"
// @Success      201 {object} models.ReviewMedia
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /reviews/{id}/media [post]
func (h *ReviewHandler) AddMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reviewID := chi.URLParam(r, "id")

	// Check ownership before touching the multipart body / disk at all.
	review, err := h.svc.Get(r.Context(), reviewID)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	if review.UserID != userID {
		writeError(w, http.StatusForbidden, "forbidden: not the review owner")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)
	if err := r.ParseMultipartForm(h.maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or malformed upload (max 25MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing \"file\" field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	spec, ok := allowedMedia[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported media type: "+contentType+" (allowed: jpeg/png/webp/heic images, mp4/mov video)")
		return
	}

	dir := filepath.Join(h.uploadDir, "reviews", reviewID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	filename := uuid.NewString() + spec.ext
	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}
	dst.Close()

	relURL := "/media/reviews/" + reviewID + "/" + filename
	media, err := h.svc.AddMedia(r.Context(), userID, reviewID, spec.kind, relURL)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

func writeReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrReviewNotFound):
		writeError(w, http.StatusNotFound, "review not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden: not the review owner")
	case errors.Is(err, service.ErrInvalidRating):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "failed to process request")
	}
}
