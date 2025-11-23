package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os/exec"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed html/*
var html embed.FS

//go:embed js/*
var js embed.FS

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var t = &Template{
	templates: template.Must(template.ParseFS(html, "html/*")),
}

var mpvCmd *exec.Cmd

func main() {
	logger, err := NewLogger()
	if err != nil {
		slog.Error("failed to initialize logger", slog.String("error", err.Error()))
		return
	}
	defer logger.logFile.Close()

	e := echo.New()
	e.Renderer = t

	e.Pre(middleware.RemoveTrailingSlash())

	e.Use(middleware.Recover())

	applyLoggingMiddleware(e, logger)

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: http.FS(js),
		IgnoreBase: true,
	}))

	data := map[string]string{
		"Title": "gifthing",
	}

	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index.html", data)
	})

	e.POST("/setgif", func(c echo.Context) error {
		var req SetGifRequest
		err := c.Bind(&req)
		if err != nil {
			return echo.NewHTTPError(400, "gifUrl not sent in body")
		}

		if mpvCmd != nil && mpvCmd.Process != nil {
			err := mpvCmd.Process.Kill()
			if err != nil {
				return echo.NewHTTPError(500, fmt.Errorf("failed to kill existing mpv process: %v", err))
			}
		}

		mpvCmd = exec.Command("mpv", "--loop=inf", "/Users/nick/andfm.mp4")
		stdout, err := mpvCmd.StdoutPipe()
		if err != nil {
			return echo.NewHTTPError(500, fmt.Errorf("failed to get mpv stdout pipe: %v", err))
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "MPV_OUTPUT", slog.Attr{Key: "mpvout", Value: slog.StringValue(scanner.Text())})
			}
		}()

		err = mpvCmd.Start()
		if err != nil {
			return echo.NewHTTPError(500, fmt.Errorf("failed to start mpv command, is the binary installed? : %v", err))
		}

		return c.JSON(200, req)
	})

	e.Start(":8080")
}
