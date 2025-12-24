package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

// extractTenorID extracts the Tenor ID from various Tenor URL formats
func extractTenorID(tenorURL string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(tenorURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Extract ID from different Tenor URL formats
	// Format 1: https://tenor.com/view/some-name-gif-ID
	// Format 2: https://media.tenor.com/path/file.gif (contains ID in path)
	// Format 3: https://tenor.com/ID.gif

	// Check if it's a tenor.com/view/ URL
	if strings.Contains(parsedURL.Path, "/view/") {
		// Extract ID from the end of the path
		parts := strings.Split(parsedURL.Path, "-")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// Remove any file extension
			id := strings.TrimSuffix(lastPart, ".gif")
			if id != "" {
				return id, nil
			}
		}
	}

	// Check if it's a media.tenor.com URL or direct ID format
	if strings.Contains(parsedURL.Host, "tenor.com") {
		// Try to extract ID using regex pattern
		re := regexp.MustCompile(`(\d{17,})`)
		matches := re.FindStringSubmatch(tenorURL)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not extract Tenor ID from URL: %s", tenorURL)
}

// getTenorMP4URL calls the Tenor API to get the MP4 URL for a given Tenor ID
func getTenorMP4URL(tenorID string) (string, error) {
	// Construct the Tenor API URL
	apiURL := fmt.Sprintf("https://tenor.googleapis.com/v2/posts?ids=%s&key=%s&client_key=gifthing", tenorID, os.Getenv("TENOR_API_KEY"))

	// Make the HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to call Tenor API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Tenor API returned status %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	var tenorResp TenorResponse
	err = json.Unmarshal(body, &tenorResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %v", err)
	}

	// Check if we have results
	if len(tenorResp.Results) == 0 {
		return "", fmt.Errorf("no results found for Tenor ID: %s", tenorID)
	}

	// Extract the MP4 URL
	mp4URL := tenorResp.Results[0].MediaFormats.MP4.URL
	if mp4URL == "" {
		return "", fmt.Errorf("no MP4 URL found in response")
	}

	return mp4URL, nil
}

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

	e.POST("/tenor-mp4", func(c echo.Context) error {
		var req TenorMp4Request
		err := c.Bind(&req)
		if err != nil {
			return echo.NewHTTPError(400, "tenorGifUrl not sent in body")
		}

		// Extract the Tenor ID from the GIF URL
		tenorID, err := extractTenorID(req.TenorGifURL)
		if err != nil {
			return echo.NewHTTPError(400, fmt.Sprintf("invalid tenor URL: %v", err))
		}

		// Call Tenor API to get the MP4 URL
		mp4URL, err := getTenorMP4URL(tenorID)
		if err != nil {
			return echo.NewHTTPError(500, fmt.Sprintf("failed to get MP4 URL: %v", err))
		}

		return c.JSON(200, map[string]string{"mp4Url": mp4URL})
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
