package handler

import (
	"family-expenses/internal/service"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	Service *service.Service
}

func RegisterRoutes(e *echo.Echo, h *Handler) {
	e.Static("/static", "frontend/static")
	e.GET("/", func(c echo.Context) error {
		return c.File("frontend/index.html")
	})

	e.GET("/expenses", h.ListExpenses)
	e.POST("/expenses", h.CreateExpense)
	e.PUT("/expenses/:id", h.UpdateExpense)
	e.DELETE("/expenses/:id", h.DeleteExpense)
}
