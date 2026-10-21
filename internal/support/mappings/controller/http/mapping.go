package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/clint456/edgex-go/internal/support/mappings/application"
	"github.com/clint456/edgex-go/internal/support/mappings/model"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/labstack/echo/v4"
)

type MappingController struct{ dic *di.Container }

func NewMappingController(dic *di.Container) *MappingController { return &MappingController{dic: dic} }

type batchRequest struct {
	NorthPoints []model.NorthPoint `json:"northPoints,omitempty"`
	SouthPoints []model.SouthPoint `json:"southPoints,omitempty"`
}

func (mc *MappingController) Add(c echo.Context) error {
	var mapping model.Mapping
	if err := c.Bind(&mapping); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	created, err := application.Add(c.Request().Context(), mapping, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusCreated, created)
}

func (mc *MappingController) Update(c echo.Context) error {
	var mapping model.Mapping
	if err := c.Bind(&mapping); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	updated, err := application.Update(c.Request().Context(), mapping, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, updated)
}

func (mc *MappingController) Delete(c echo.Context) error {
	if err := application.Delete(c.Request().Context(), c.Param("id"), mc.dic); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (mc *MappingController) DeleteByNorthNodeName(c echo.Context) error {
	if err := application.DeleteByNorthNodeName(c.Request().Context(), c.Param("northNodeName"), mc.dic); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (mc *MappingController) DeleteByNorthProfileName(c echo.Context) error {
	if err := application.DeleteByNorthProfileName(c.Request().Context(), c.Param("northProfileName"), mc.dic); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (mc *MappingController) DeleteBySouthNodeName(c echo.Context) error {
	if err := application.DeleteBySouthNodeName(c.Request().Context(), c.Param("southNodeName"), mc.dic); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (mc *MappingController) DeleteBySouthProfileName(c echo.Context) error {
	if err := application.DeleteBySouthProfileName(c.Request().Context(), c.Param("southProfileName"), mc.dic); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (mc *MappingController) Forward(c echo.Context) error {
	point := model.NorthPoint{
		NorthNodeName: c.QueryParam("northNodeName"), NorthProfileName: c.QueryParam("northProfileName"),
		NorthResourceName: c.QueryParam("northResourceName"),
	}
	mapping, err := application.Forward(c.Request().Context(), point, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, mapping)
}

func (mc *MappingController) Reverse(c echo.Context) error {
	point := model.SouthPoint{
		SouthNodeName: c.QueryParam("southNodeName"), SouthProfileName: c.QueryParam("southProfileName"),
		SouthResourceName: c.QueryParam("southResourceName"),
	}
	mapping, err := application.Reverse(c.Request().Context(), point, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, mapping)
}

func (mc *MappingController) Batch(c echo.Context) error {
	var request batchRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	mappings, err := application.Batch(c.Request().Context(), c.Param("direction"), request.NorthPoints, request.SouthPoints, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, mappings)
}

func (mc *MappingController) ByNorthNodeName(c echo.Context) error {
	offset, limit, err := pagination(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	mappings, total, err := application.ByNorthNodeName(c.Request().Context(), c.Param("northNodeName"), offset, limit, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"totalCount": total, "mappings": mappings})
}

func (mc *MappingController) ByNorthProfileName(c echo.Context) error {
	offset, limit, err := pagination(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	mappings, total, err := application.ByNorthProfileName(c.Request().Context(), c.Param("northProfileName"), offset, limit, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"totalCount": total, "mappings": mappings})
}

func (mc *MappingController) BySouthNodeName(c echo.Context) error {
	offset, limit, err := pagination(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	mappings, total, err := application.BySouthNodeName(c.Request().Context(), c.Param("southNodeName"), offset, limit, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"totalCount": total, "mappings": mappings})
}

func (mc *MappingController) BySouthProfileName(c echo.Context) error {
	offset, limit, err := pagination(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	mappings, total, err := application.BySouthProfileName(c.Request().Context(), c.Param("southProfileName"), offset, limit, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"totalCount": total, "mappings": mappings})
}

func (mc *MappingController) All(c echo.Context) error {
	offset, limit, err := pagination(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
	}
	mappings, total, err := application.All(c.Request().Context(), offset, limit, mc.dic)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"totalCount": total, "mappings": mappings})
}

func pagination(c echo.Context) (int, int, error) {
	offset, limit := 0, 100
	var err error
	if raw := c.QueryParam("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("offset must be an integer")
		}
	}
	if raw := c.QueryParam("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("limit must be an integer")
		}
	}
	return offset, limit, nil
}

type errorResponse struct {
	Message string `json:"message"`
}

func writeError(c echo.Context, err error) error {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrInvalidMapping):
		status = http.StatusBadRequest
	case errors.Is(err, application.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, application.ErrDuplicate):
		status = http.StatusConflict
	}
	return c.JSON(status, errorResponse{err.Error()})
}
