package http

import (
    "github.com/labstack/echo/v4"
    "net/http"
    "github.com/kriipke/yiff/internal/app"
)

func StartServer() {
    e := echo.New()
    e.POST("/diff", func(c echo.Context) error {
        a := c.FormValue("a")
        b := c.FormValue("b")
        result, _ := app.DiffYAML([]byte(a), []byte(b))
        return c.JSON(http.StatusOK, result)
    })
    e.Logger.Fatal(e.Start(":8080"))
}
