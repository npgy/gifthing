package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

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
var mpvLock = sync.Mutex{}

var rootMp4Path = "/root/"
var testMode = false

func main() {
	if os.Getenv("TESTMODE") == "true" {
		rootMp4Path = "/Users/nick/"
		testMode = true
	}
	logger, err := NewLogger()
	if err != nil {
		slog.Error("failed to initialize logger", slog.String("error", err.Error()))
		return
	}
	defer logger.logFile.Close()

	mpvCmd, err = startMpv("main.mp4", logger)
	if err != nil {
		slog.Error("failed to start mpv", slog.String("error", err.Error()))
		return
	}

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

		err = killMpv(logger)
		if err != nil {
			return err
		}

		showLoadingScreen(logger)
		time.Sleep(3000 * time.Millisecond)

		go func() {
			curlCmd := exec.Command("curl", req.GifURL, "-o", rootMp4Path+"main.mp4")
			stdout, err := curlCmd.StdoutPipe()
			if err != nil {
				logger.LogSimple(slog.LevelError, "CURL_OUTPUT", fmt.Sprintf("error getting curl stdout: %v", err))
				showErrorScreen(logger)
				return
			}

			logger.LogSimple(slog.LevelInfo, "CURL_OUTPUT", fmt.Sprintf("beginning to download %s", req.GifURL))

			ctx, cancel := context.WithCancel(context.Background())
			go pumpStdOut(ctx, cancel, stdout, "CURL_OUTPUT", logger)

			err = curlCmd.Start()
			if err != nil {
				logger.LogSimple(slog.LevelError, "CURL_OUTPUT", fmt.Sprintf("error starting curl command: %v", err))
				showErrorScreen(logger)
				return
			}

			err = killMpv(logger)
			if err != nil {
				logger.LogSimple(slog.LevelError, "MPV_OUTPUT", fmt.Sprintf("error killing mpv: %v", err))
				return
			}

			mpvLock.Lock()
			defer mpvLock.Unlock()
			mpvCmd, err = startMpv("main.mp4", logger)
			// monitorMpvExit(mpvCmd, logger)
			// if err != nil {
			// 	logger.LogSimple(slog.LevelError, "MPV_OUTPUT", fmt.Sprintf("error starting mpv on main.mp4: %v", err))
			// 	showErrorScreen(logger)
			// 	return
			// }
		}()

		return c.NoContent(201)
	})

	port := 80
	if testMode {
		port = 8080
	}
	e.Start(fmt.Sprintf(":%d", port))
}
