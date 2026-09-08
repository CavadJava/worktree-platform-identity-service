package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"platform-identity-service/internal/middleware"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type createProductRequest struct {
	Name string `json:"name"`
}

type productResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TechStack   string `json:"tech_stack"`
	CreatedAt   string `json:"created_at"`
}

func toProductResponse(p *models.Product) productResponse {
	return productResponse{
		ID: p.ID, Name: p.Name, Description: p.Description, TechStack: p.TechStack,
		CreatedAt: p.CreatedAt.Format(timeFormat),
	}
}

// Create godoc
// @Summary      Create a product
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body createProductRequest true "Product payload"
// @Success      201 {object} productResponse
// @Failure      403 {object} map[string]string
// @Router       /products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if caller.SystemRole != models.SystemRoleSuperadmin {
		writeError(w, http.StatusForbidden, "superadmin role required")
		return
	}

	var req createProductRequest
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
		writeError(w, http.StatusInternalServerError, "failed to create product")
		return
	}
	writeJSON(w, http.StatusCreated, toProductResponse(p))
}

// List godoc
// @Summary      List all products
// @Tags         products
// @Produce      json
// @Success      200 {array} productResponse
// @Router       /products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	response := make([]productResponse, len(products))
	for i, p := range products {
		response[i] = toProductResponse(&p)
	}
	writeJSON(w, http.StatusOK, response)
}

type accessResponse struct {
	Access string `json:"access"`
}

// CheckAccess godoc
// @Summary      Check the caller's access level to a product
// @Description  Returns "full" if subscribed, "demo" otherwise.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      200 {object} accessResponse
// @Failure      404 {object} map[string]string
// @Router       /products/{id}/access [get]
func (h *ProductHandler) CheckAccess(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	access, err := h.svc.CheckAccess(r.Context(), caller.UserID, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check access")
		return
	}
	writeJSON(w, http.StatusOK, accessResponse{Access: access})
}

type setSubscriptionRequest struct {
	Subscripted bool   `json:"subscripted"`
	Renewed     bool   `json:"renewed"`
	Notes       string `json:"notes"`
}

type subscriptionResponse struct {
	UserID      string `json:"user_id"`
	ProductID   string `json:"product_id"`
	Subscripted bool   `json:"subscripted"`
	Renewed     bool   `json:"renewed"`
	Notes       string `json:"notes"`
}

// SetSubscription godoc
// @Summary      Manually set a user's subscription to a product
// @Description  Superadmin, or an admin who manages this product. No payment gateway integration.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID"
// @Param        productId path string true "Product ID"
// @Param        request body setSubscriptionRequest true "Subscription payload"
// @Success      200 {object} subscriptionResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{userId}/products/{productId}/subscribe [post]
func (h *ProductHandler) SetSubscription(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	targetUserID := chi.URLParam(r, "userId")
	productID := chi.URLParam(r, "productId")
	sub, err := h.svc.SetSubscription(r.Context(), caller, targetUserID, productID, req.Subscripted, req.Renewed, req.Notes)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to set subscription")
		}
		return
	}
	writeJSON(w, http.StatusOK, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed, Notes: sub.Notes,
	})
}

type updateProductProfileRequest struct {
	Description string `json:"description"`
	TechStack   string `json:"tech_stack"`
}

// UpdateProfile godoc
// @Summary      Set a product's description and tech stack
// @Description  Superadmin, or an admin who manages this product.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body updateProductProfileRequest true "Profile payload"
// @Success      200 {object} productResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/profile [post]
func (h *ProductHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProductProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	productID := chi.URLParam(r, "id")
	p, err := h.svc.UpdateProfile(r.Context(), caller, productID, req.Description, req.TechStack)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to update profile")
		}
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(p))
}

type addSubprojectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type subprojectResponse struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// AddSubproject godoc
// @Summary      Add a named subproject/module to a product
// @Description  Superadmin, or an admin who manages this product.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body addSubprojectRequest true "Subproject payload"
// @Success      201 {object} subprojectResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/subprojects [post]
func (h *ProductHandler) AddSubproject(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addSubprojectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	productID := chi.URLParam(r, "id")
	sub, err := h.svc.AddSubproject(r.Context(), caller, productID, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to add subproject")
		}
		return
	}
	writeJSON(w, http.StatusCreated, subprojectResponse{
		ID: sub.ID, ProductID: sub.ProductID, Name: sub.Name, Description: sub.Description,
		CreatedAt: sub.CreatedAt.Format(timeFormat),
	})
}

// ListSubprojects godoc
// @Summary      List a product's subprojects
// @Tags         products
// @Produce      json
// @Param        id path string true "Product ID"
// @Success      200 {array} subprojectResponse
// @Router       /products/{id}/subprojects [get]
func (h *ProductHandler) ListSubprojects(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	subprojects, err := h.svc.ListSubprojects(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subprojects")
		return
	}

	response := make([]subprojectResponse, len(subprojects))
	for i, s := range subprojects {
		response[i] = subprojectResponse{
			ID: s.ID, ProductID: s.ProductID, Name: s.Name, Description: s.Description,
			CreatedAt: s.CreatedAt.Format(timeFormat),
		}
	}
	writeJSON(w, http.StatusOK, response)
}

// RemoveSubproject godoc
// @Summary      Remove a subproject
// @Description  Superadmin, or an admin who manages this product.
// @Tags         products
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        subId path string true "Subproject ID"
// @Success      204
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/subprojects/{subId} [delete]
func (h *ProductHandler) RemoveSubproject(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subID := chi.URLParam(r, "subId")
	err := h.svc.RemoveSubproject(r.Context(), caller, subID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, repository.ErrSubprojectNotFound):
			writeError(w, http.StatusNotFound, "subproject not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to remove subproject")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createProductUserRequest struct {
	Name       string `json:"name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	SystemRole string `json:"system_role"`
}

// CreateUserAndSubscribe godoc
// @Summary      Create a brand-new user and subscribe it to a product
// @Description  Superadmin, or an admin who manages this product. Creates
// @Description  an account with the given system role ('user' or 'admin')
// @Description  and immediately marks it subscribed to the product.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body createProductUserRequest true "New user payload"
// @Success      201 {object} subscriptionResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/users [post]
func (h *ProductHandler) CreateUserAndSubscribe(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createProductUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, username, email and password are required")
		return
	}
	systemRole := req.SystemRole
	if systemRole == "" {
		systemRole = models.SystemRoleUser
	}
	if systemRole != models.SystemRoleUser && systemRole != models.SystemRoleAdmin {
		writeError(w, http.StatusBadRequest, "system_role must be 'user' or 'admin'")
		return
	}

	productID := chi.URLParam(r, "id")
	sub, err := h.svc.CreateUserAndSubscribe(r.Context(), caller, productID, service.CreateUserInput{
		Name: req.Name, Username: req.Username, Email: req.Email, Password: req.Password,
	}, systemRole)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, service.ErrUsernameTaken):
			writeError(w, http.StatusConflict, "username already registered")
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}
	writeJSON(w, http.StatusCreated, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed, Notes: sub.Notes,
	})
}

// ListMine godoc
// @Summary      List products the caller may administer
// @Description  Superadmin sees every product; admin sees only products they hold a subscription to.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} productResponse
// @Router       /products/mine [get]
func (h *ProductHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	products, err := h.svc.ListForCaller(r.Context(), caller)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	response := make([]productResponse, len(products))
	for i, p := range products {
		response[i] = toProductResponse(&p)
	}
	writeJSON(w, http.StatusOK, response)
}

type productBrowseResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListBrowse godoc
// @Summary      Read-only list of every product's id and name
// @Description  Any authenticated caller — lets an admin with no subscriptions see what products exist.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} productBrowseResponse
// @Router       /products/browse [get]
func (h *ProductHandler) ListBrowse(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	response := make([]productBrowseResponse, len(products))
	for i, p := range products {
		response[i] = productBrowseResponse{ID: p.ID, Name: p.Name}
	}
	writeJSON(w, http.StatusOK, response)
}

type requestAdminRequest struct {
	SubjectUserID string `json:"subject_user_id"`
}

type productAdminRequestResponse struct {
	ID                string `json:"id"`
	ProductID         string `json:"product_id"`
	SubjectUserID     string `json:"subject_user_id"`
	RequestedByUserID string `json:"requested_by_user_id"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
}

func toProductAdminRequestResponse(req *models.ProductAdminRequest) productAdminRequestResponse {
	return productAdminRequestResponse{
		ID: req.ID, ProductID: req.ProductID, SubjectUserID: req.SubjectUserID,
		RequestedByUserID: req.RequestedByUserID, Status: req.Status,
		CreatedAt: req.CreatedAt.Format(timeFormat),
	}
}

// RequestAdmin godoc
// @Summary      Request that a user become admin for a product
// @Description  Admin (managing this product) or superadmin. Requires superadmin approval to take effect.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body requestAdminRequest true "Subject payload"
// @Success      201 {object} productAdminRequestResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/admin-requests [post]
func (h *ProductHandler) RequestAdmin(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req requestAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectUserID == "" {
		writeError(w, http.StatusBadRequest, "subject_user_id is required")
		return
	}

	productID := chi.URLParam(r, "id")
	created, err := h.svc.RequestProductAdmin(r.Context(), caller, productID, req.SubjectUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, service.ErrAlreadyHasAccess):
			writeError(w, http.StatusConflict, "subject already has access to this product")
		case errors.Is(err, service.ErrRequestAlreadyPending):
			writeError(w, http.StatusConflict, "a pending request already exists for this user and product")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create request")
		}
		return
	}
	writeJSON(w, http.StatusCreated, toProductAdminRequestResponse(created))
}

// ListAdminRequests godoc
// @Summary      List pending admin requests for a product
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      200 {array} productAdminRequestResponse
// @Router       /products/{id}/admin-requests [get]
func (h *ProductHandler) ListAdminRequests(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	requests, err := h.svc.ListPendingRequests(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list requests")
		return
	}
	response := make([]productAdminRequestResponse, len(requests))
	for i, req := range requests {
		response[i] = toProductAdminRequestResponse(&req)
	}
	writeJSON(w, http.StatusOK, response)
}

type decideAdminRequestRequest struct {
	Approve bool `json:"approve"`
}

// DecideAdminRequest godoc
// @Summary      Approve or reject a pending admin request
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        requestId path string true "Request ID"
// @Param        request body decideAdminRequestRequest true "Decision payload"
// @Success      200 {object} productAdminRequestResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/admin-requests/{requestId}/decide [post]
func (h *ProductHandler) DecideAdminRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req decideAdminRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	requestID := chi.URLParam(r, "requestId")
	decided, err := h.svc.DecideRequest(r.Context(), caller, requestID, req.Approve)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		case errors.Is(err, repository.ErrProductAdminRequestNotFound):
			writeError(w, http.StatusNotFound, "request not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to decide request")
		}
		return
	}
	writeJSON(w, http.StatusOK, toProductAdminRequestResponse(decided))
}

type promoteAdminRequest struct {
	SubjectUserID string `json:"subject_user_id"`
}

// PromoteAdmin godoc
// @Summary      Directly promote a user to admin for a product
// @Description  Superadmin only. No request is created — immediate effect.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body promoteAdminRequest true "Subject payload"
// @Success      200 {object} subscriptionResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/admin [post]
func (h *ProductHandler) PromoteAdmin(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req promoteAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectUserID == "" {
		writeError(w, http.StatusBadRequest, "subject_user_id is required")
		return
	}

	productID := chi.URLParam(r, "id")
	sub, err := h.svc.PromoteDirectly(r.Context(), caller, productID, req.SubjectUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to promote user")
		}
		return
	}
	writeJSON(w, http.StatusOK, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed, Notes: sub.Notes,
	})
}

type customerResponse struct {
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	SystemRole  string `json:"system_role"`
	Subscripted bool   `json:"subscripted"`
	Renewed     bool   `json:"renewed"`
	Notes       string `json:"notes"`
}

// ListCustomers godoc
// @Summary      List every subscriber of a product
// @Description  Superadmin, or an admin who manages this product.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      200 {array} customerResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/customers [get]
func (h *ProductHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	customers, err := h.svc.ListCustomers(r.Context(), caller, productID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		default:
			writeError(w, http.StatusInternalServerError, "failed to list customers")
		}
		return
	}

	response := make([]customerResponse, len(customers))
	for i, c := range customers {
		response[i] = customerResponse{
			UserID: c.UserID, Name: c.UserName, Username: c.UserUsername, Email: c.UserEmail,
			SystemRole: c.UserSystemRole, Subscripted: c.Subscripted, Renewed: c.Renewed, Notes: c.Notes,
		}
	}
	writeJSON(w, http.StatusOK, response)
}
