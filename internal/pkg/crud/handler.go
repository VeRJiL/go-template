package crud

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/VeRJiL/go-template/internal/pkg/logger"
	"github.com/VeRJiL/go-template/internal/pkg/modules"
)

// GenericHandler implements the Handler interface for any entity
type GenericHandler[T modules.Entity] struct {
	service    modules.Service[T]
	logger     *logger.Logger
	entityName string
}

// NewGenericHandler creates a new generic handler
func NewGenericHandler[T modules.Entity](service modules.Service[T], logger *logger.Logger, entityName string) *GenericHandler[T] {
	return &GenericHandler[T]{
		service:    service,
		logger:     logger,
		entityName: entityName,
	}
}

// Create handles POST requests to create a new entity
// @Summary Create a new entity
// @Description Create a new entity with the provided data
// @Tags entities
// @Accept json
// @Produce json
// @Param entity body object true "Entity data"
// @Success 201 {object} object "Created entity"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /entities [post]
func (h *GenericHandler[T]) Create(c *gin.Context) {
	var entity T
	if err := c.ShouldBindJSON(&entity); err != nil {
		h.logger.Error("Failed to bind JSON", "error", err, "entity", h.entityName)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	createdEntity, err := h.service.Create(c.Request.Context(), &entity)
	if err != nil {
		h.logger.Error("Failed to create entity", "error", err, "entity", h.entityName)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create " + h.entityName,
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Entity created successfully", "id", (*createdEntity).GetID(), "entity", h.entityName)
	c.JSON(http.StatusCreated, SuccessResponse{
		Message: h.entityName + " created successfully",
		Data:    createdEntity,
	})
}

// GetByID handles GET requests to retrieve an entity by ID
// @Summary Get entity by ID
// @Description Retrieve a specific entity by its ID
// @Tags entities
// @Produce json
// @Param id path int true "Entity ID"
// @Success 200 {object} object "Entity data"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Entity not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /entities/{id} [get]
func (h *GenericHandler[T]) GetByID(c *gin.Context) {
	id, err := h.getIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID parameter",
			Message: err.Error(),
		})
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get entity", "error", err, "id", id, "entity", h.entityName)
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   h.entityName + " not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: h.entityName + " retrieved successfully",
		Data:    entity,
	})
}

// Update handles PUT requests to update an entity
// @Summary Update entity
// @Description Update an existing entity with the provided data
// @Tags entities
// @Accept json
// @Produce json
// @Param id path int true "Entity ID"
// @Param entity body object true "Updated entity data"
// @Success 200 {object} object "Updated entity"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Entity not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /entities/{id} [put]
func (h *GenericHandler[T]) Update(c *gin.Context) {
	id, err := h.getIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID parameter",
			Message: err.Error(),
		})
		return
	}

	var entity T
	if err := c.ShouldBindJSON(&entity); err != nil {
		h.logger.Error("Failed to bind JSON", "error", err, "entity", h.entityName)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), id, &entity)
	if err != nil {
		h.logger.Error("Failed to update entity", "error", err, "id", id, "entity", h.entityName)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update " + h.entityName,
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Entity updated successfully", "id", id, "entity", h.entityName)
	c.JSON(http.StatusOK, SuccessResponse{
		Message: h.entityName + " updated successfully",
		Data:    updatedEntity,
	})
}

// Delete handles DELETE requests to remove an entity
// @Summary Delete entity
// @Description Delete an entity by its ID
// @Tags entities
// @Produce json
// @Param id path int true "Entity ID"
// @Success 200 {object} SuccessResponse "Entity deleted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Entity not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /entities/{id} [delete]
func (h *GenericHandler[T]) Delete(c *gin.Context) {
	id, err := h.getIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid ID parameter",
			Message: err.Error(),
		})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("Failed to delete entity", "error", err, "id", id, "entity", h.entityName)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete " + h.entityName,
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Entity deleted successfully", "id", id, "entity", h.entityName)
	c.JSON(http.StatusOK, SuccessResponse{
		Message: h.entityName + " deleted successfully",
		Data:    nil,
	})
}

// List handles GET requests to list entities with filtering and pagination
// @Summary List entities
// @Description Retrieve a list of entities with optional filtering and pagination
// @Tags entities
// @Produce json
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(20)
// @Param search query string false "Search term"
// @Param sort_by query string false "Field to sort by"
// @Param sort_order query string false "Sort order (asc or desc)" default(asc)
// @Success 200 {object} PaginationResponse "List of entities"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /entities [get]
func (h *GenericHandler[T]) List(c *gin.Context) {
	filters := h.parseListFilters(c)

	entities, total, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to list entities", "error", err, "entity", h.entityName)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list " + h.entityName + "s",
			Message: err.Error(),
		})
		return
	}

	// Calculate total pages
	totalPages := int(total) / filters.Limit
	if int(total)%filters.Limit != 0 {
		totalPages++
	}

	response := modules.PaginationResponse{
		Data:       entities,
		Total:      total,
		Offset:     filters.Offset,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: h.entityName + "s retrieved successfully",
		Data:    response,
	})
}

// Helper methods

func (h *GenericHandler[T]) getIDFromParam(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func (h *GenericHandler[T]) parseListFilters(c *gin.Context) modules.ListFilters {
	// Default values
	filters := modules.ListFilters{
		Offset:    0,
		Limit:     20,
		SortOrder: "asc",
		Filters:   make(map[string]string),
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filters.Limit = limit
		}
	}

	// Parse search
	filters.Search = c.Query("search")

	// Parse sort parameters
	filters.SortBy = c.Query("sort_by")
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		if sortOrder == "desc" || sortOrder == "asc" {
			filters.SortOrder = sortOrder
		}
	}

	// Parse custom filters (any query parameter not handled above)
	for key, values := range c.Request.URL.Query() {
		if key != "offset" && key != "limit" && key != "search" && key != "sort_by" && key != "sort_order" {
			if len(values) > 0 {
				filters.Filters[key] = values[0]
			}
		}
	}

	return filters
}

// Response types

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Additional handler methods for advanced operations

// BulkCreate handles POST requests to create multiple entities
func (h *GenericHandler[T]) BulkCreate(c *gin.Context) {
	var entities []*T
	if err := c.ShouldBindJSON(&entities); err != nil {
		h.logger.Error("Failed to bind JSON for bulk create", "error", err, "entity", h.entityName)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Cast to interface that supports BulkCreate
	if bulkService, ok := h.service.(interface {
		BulkCreate(context.Context, []*T) ([]*T, error)
	}); ok {
		createdEntities, err := bulkService.BulkCreate(c.Request.Context(), entities)
		if err != nil {
			h.logger.Error("Failed to bulk create entities", "error", err, "entity", h.entityName)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to bulk create " + h.entityName + "s",
				Message: err.Error(),
			})
			return
		}

		h.logger.Info("Entities bulk created successfully", "count", len(createdEntities), "entity", h.entityName)
		c.JSON(http.StatusCreated, SuccessResponse{
			Message: h.entityName + "s bulk created successfully",
			Data:    createdEntities,
		})
	} else {
		c.JSON(http.StatusNotImplemented, ErrorResponse{
			Error:   "Bulk create not supported",
			Message: "This entity does not support bulk creation",
		})
	}
}

// Count handles GET requests to count entities
func (h *GenericHandler[T]) Count(c *gin.Context) {
	filters := h.parseListFilters(c)

	// Cast to interface that supports Count
	if countService, ok := h.service.(interface {
		Count(context.Context, modules.ListFilters) (int64, error)
	}); ok {
		count, err := countService.Count(c.Request.Context(), filters)
		if err != nil {
			h.logger.Error("Failed to count entities", "error", err, "entity", h.entityName)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to count " + h.entityName + "s",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Message: h.entityName + " count retrieved successfully",
			Data:    map[string]int64{"count": count},
		})
	} else {
		c.JSON(http.StatusNotImplemented, ErrorResponse{
			Error:   "Count not supported",
			Message: "This entity does not support counting",
		})
	}
}