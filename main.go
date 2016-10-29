package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
)

func getUser(c echo.Context) error {
	// User ID from path `users/:id`
	id := c.Param("id")
	return c.String(http.StatusOK, "Hello, "+id)
}

func main() {
	e := echo.New()
	e.GET("/users/:id", getUser)
	e.Run(standard.New(":5000"))
}
