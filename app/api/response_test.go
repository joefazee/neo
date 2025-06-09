package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAPIResponses(t *testing.T) {
	t.Run("SuccessResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := map[string]string{"key": "value"}
		SuccessResponse(c, http.StatusOK, "Success message", data)

		assert.Equal(t, http.StatusOK, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Success message", response.Message)
		assert.NotNil(t, response.Data)
		assert.Nil(t, response.Error)
	})

	t.Run("SuccessResponseWithMeta", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := []string{"item1", "item2"}
		meta := PaginationMeta{Page: 1, PerPage: 10}
		SuccessResponseWithMeta(c, http.StatusOK, "Success with meta", data, meta)

		assert.Equal(t, http.StatusOK, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Success with meta", response.Message)
		assert.NotNil(t, response.Data)
		assert.NotNil(t, response.Meta)
		assert.Nil(t, response.Error)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		details := map[string]string{"field": "error"}
		ErrorResponse(c, http.StatusBadRequest, "TEST_ERROR", "Test error message", details)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.NotNil(t, response.Error)
		assert.Equal(t, "TEST_ERROR", response.Error.Code)
		assert.Equal(t, "Test error message", response.Error.Message)
		assert.NotNil(t, response.Error.Details)
	})

	t.Run("ValidationErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		details := "Invalid email format"
		BadRequestResponse(c, details)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "BAD_REQUEST", response.Error.Code)
		assert.Equal(t, "Invalid request data", response.Error.Message)
	})

	t.Run("NotFoundResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		NotFoundResponse(c, "User")

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "NOT_FOUND", response.Error.Code)
		assert.Equal(t, "User not found", response.Error.Message)
	})

	t.Run("UnauthorizedResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		UnauthorizedResponse(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "UNAUTHORIZED", response.Error.Code)
		assert.Equal(t, "Unauthorized access", response.Error.Message)
	})

	t.Run("ForbiddenResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		ForbiddenResponse(c, "Access denied")

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "FORBIDDEN", response.Error.Code)
		assert.Equal(t, "Access denied", response.Error.Message)
	})

	t.Run("InternalErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		InternalErrorResponse(c, "Database connection failed")

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
		assert.Equal(t, "Database connection failed", response.Error.Message)
	})

	t.Run("ConflictResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		ConflictResponse(c, "Email already exists")

		assert.Equal(t, http.StatusConflict, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "CONFLICT", response.Error.Code)
		assert.Equal(t, "Email already exists", response.Error.Message)
	})

	t.Run("CreatedResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := map[string]string{"id": "123"}
		CreatedResponse(c, "Resource created", data)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Resource created", response.Message)
	})

	t.Run("UpdatedResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := map[string]string{"id": "123"}
		UpdatedResponse(c, "Resource updated", data)

		assert.Equal(t, http.StatusOK, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Resource updated", response.Message)
	})

	t.Run("DeletedResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		DeletedResponse(c, "Resource deleted")

		assert.Equal(t, http.StatusOK, w.Code)

		var response Response
		b := w.Body.Bytes()
		err := json.Unmarshal(b, &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Resource deleted", response.Message)
		assert.Nil(t, response.Data)
	})

	t.Run("ListResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := []string{"item1", "item2", "item3"}
		ListResponse(c, "Items retrieved", data, 3)

		assert.Equal(t, http.StatusOK, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Items retrieved", response.Message)
		assert.NotNil(t, response.Meta)

		metaBytes, _ := json.Marshal(response.Meta)
		var listMeta ListMeta
		json.Unmarshal(metaBytes, &listMeta)
		assert.Equal(t, 3, listMeta.Count)
	})

	t.Run("PaginatedResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := []string{"item1", "item2"}
		meta := PaginationMeta{
			Page:       1,
			PerPage:    2,
			Total:      10,
			TotalPages: 5,
			HasNext:    true,
			HasPrev:    false,
		}
		PaginatedResponse(c, "Paginated results", data, meta)

		assert.Equal(t, http.StatusOK, w.Code)

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Paginated results", response.Message)
		assert.NotNil(t, response.Meta)

		metaBytes, _ := json.Marshal(response.Meta)
		var paginationMeta PaginationMeta
		json.Unmarshal(metaBytes, &paginationMeta)
		assert.Equal(t, 1, paginationMeta.Page)
		assert.Equal(t, 2, paginationMeta.PerPage)
		assert.Equal(t, int64(10), paginationMeta.Total)
		assert.True(t, paginationMeta.HasNext)
		assert.False(t, paginationMeta.HasPrev)
	})
}
