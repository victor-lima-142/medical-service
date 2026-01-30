package request

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type PaginatedResponse struct {
	Success bool           `json:"success"`
	Data    interface{}    `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func RespondSuccess(c *gin.Context, status int, data interface{}, message string) {
	c.JSON(status, NewSuccessResponse(data, message))
}

func RespondError(c *gin.Context, status int, err, message, code string) {
	c.JSON(status, NewErrorResponse(err, message, code))
}

func RespondPaginated(c *gin.Context, data interface{}, page, perPage int, total int64) {
	c.JSON(http.StatusOK, NewPaginatedResponse(data, page, perPage, total))
}

func NewSuccessResponse(data interface{}, message string) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}
}

func NewErrorResponse(err, message, code string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   err,
		Message: message,
		Code:    code,
	}
}

func NewPaginationMeta(page, perPage int, total int64) PaginationMeta {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return PaginationMeta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

func NewPaginatedResponse(data interface{}, page, perPage int, total int64) PaginatedResponse {
	return PaginatedResponse{
		Success: true,
		Data:    data,
		Meta:    NewPaginationMeta(page, perPage, total),
	}
}
