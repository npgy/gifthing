package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"syscall"

	"github.com/labstack/echo/v4"
)

func startMpv(fileName string, logger *Logger) (*exec.Cmd, error) {
	cmd := exec.Command("mpv", "--loop=inf", rootMp4Path+fileName)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Completely detach stdio
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return cmd, echo.NewHTTPError(500, fmt.Errorf("failed to get mpv stdout pipe: %v", err))
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)

	// go pumpStdOut(ctx, cancel, stdout, "MPV_OUTPUT", logger)

	logger.LogSimple(slog.LevelInfo, "MPV_OUTPUT", fmt.Sprintf("player file %s", fileName))

	err := cmd.Start()

	if err != nil {
		return cmd, echo.NewHTTPError(500, fmt.Errorf("failed to start mpv command, is the binary installed? : %v", err))
	}

	err = cmd.Process.Release()
	if err != nil {
		logger.LogSimple(slog.LevelWarn, "MPV_OUTPUT", fmt.Sprintf("failed to release mpv process: %v", err))
	}

	return cmd, err
}

func killMpv(logger *Logger) error {
	cmd := exec.Command("killall", "-9", "mpv")
	err := cmd.Run()
	logger.LogSimple(slog.LevelInfo, "MPV_OUTPUT", "killing mpv via killall")

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			if exitCode == 1 {
				logger.LogSimple(slog.LevelInfo, "MPV_OUTPUT", "no existing mpv process to kill")
				return nil
			}
		}
		return echo.NewHTTPError(500, fmt.Errorf("failed to kill existing mpv process: %v", err))
	}

	return nil
}

func showErrorScreen(logger *Logger) {
	var err error
	mpvLock.Lock()
	defer mpvLock.Unlock()
	mpvCmd, err = startMpv("tryagain.mp4", logger)
	if err != nil {
		logger.LogSimple(slog.LevelError, "MPV_OUTPUT", fmt.Sprintf("error showing the error screen: %v", err))
	}
}

func showLoadingScreen(logger *Logger) {
	var err error
	mpvLock.Lock()
	defer mpvLock.Unlock()
	mpvCmd, err = startMpv("loadinggif.mp4", logger)
	if err != nil {
		logger.LogSimple(slog.LevelError, "MPV_OUTPUT", fmt.Sprintf("error showing loading animation: %v", err))
	}
}

func pumpStdOut(ctx context.Context, cancel context.CancelFunc, output io.ReadCloser, logMsg string, logger *Logger) {
	defer cancel()
	scanner := bufio.NewScanner(output)

	for {
		select {
		case <-ctx.Done():
			logger.LogSimple(slog.LevelInfo, logMsg, fmt.Sprintf("pumpStdOut stopped: %v", ctx.Err()))
			return
		default:
			if scanner.Scan() {
				logger.LogSimple(slog.LevelInfo, logMsg, scanner.Text())
			} else {
				// Scanner finished or error occurred
				if err := scanner.Err(); err != nil {
					logger.LogSimple(slog.LevelError, logMsg, fmt.Sprintf("scanner error: %v", err))
				}
				return
			}
		}
	}
}

func monitorMpvExit(cmd *exec.Cmd, logger *Logger) {
	go func() {
		err := cmd.Wait() // Blocks until process exits

		if err != nil {
			logger.LogSimple(slog.LevelError, "MPV_EXIT", fmt.Sprintf("mpv exited with error: %v", err))
			// Auto-restart with error screen
			showErrorScreen(logger)
		} else {
			logger.LogSimple(slog.LevelInfo, "MPV_EXIT", "mpv exited normally")
		}
	}()
}
