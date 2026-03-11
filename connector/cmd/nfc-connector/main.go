package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"web-nfc-bridge/connector/internal/api"
	"web-nfc-bridge/connector/internal/bridge"
)

var version = "dev"
var buildTime = "unknown"

const defaultAllowedOrigins = "http://localhost:*,https://localhost:*,http://127.0.0.1:*,https://127.0.0.1:*,https://web-nfc-bridge.abcd854884.workers.dev,https://web-nfc-bridge.abcd854884.workers.dev.,https://nfc.yudefine.com.tw,https://nfc.yudefine.com.tw."

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--watchdog" {
		runWatchdog()
		return
	}

	addr := getenv("NFC_CONNECTOR_ADDR", "127.0.0.1:42619")
	secret := getenv("NFC_CONNECTOR_SHARED_SECRET", "development-shared-secret")
	allowedOrigins := strings.Split(getenv("NFC_CONNECTOR_ALLOWED_ORIGINS", defaultAllowedOrigins), ",")

	var driver bridge.Driver
	pcscDriver, pcscErr := bridge.NewPCSCDriver()
	if pcscErr == nil {
		health := pcscDriver.Health(context.Background())
		if health["status"] == "ok" {
			driver = pcscDriver
		} else {
			log.Printf("pcsc driver degraded (status=%v), trying direct driver", health["status"])
		}
	} else {
		log.Printf("pcsc driver unavailable: %v", pcscErr)
	}

	if driver == nil {
		directDriver, directErr := bridge.NewDirectDriver()
		if directErr == nil {
			driver = directDriver
			if pcscDriver != nil {
				pcscDriver.Close()
			}
		} else if pcscDriver != nil {
			log.Printf("direct driver unavailable: %v; using degraded pcsc driver (will recover when pcscd starts)", directErr)
			driver = pcscDriver
		} else {
			log.Fatalf("no working driver available: pcsc=%v direct=%v", pcscErr, directErr)
		}
	}
	defer driver.Close()

	service := bridge.NewService(driver)
	server := api.NewServer(service, allowedOrigins, secret, version, buildTime)

	httpServer := &http.Server{Addr: addr, Handler: server.Handler()}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	log.Printf("nfc connector listening on http://%s (driver=%s version=%s buildTime=%s)", addr, service.DriverName(), version, buildTime)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func runWatchdog() {
	if runtime.GOOS != "windows" {
		log.Fatal("--watchdog is only supported on Windows; use launchd (macOS) or systemd (Linux) instead")
	}

	const restartDelay = 3 * time.Second
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("watchdog: cannot resolve executable path: %v", err)
	}

	log.Printf("watchdog: supervising %s", exe)
	for {
		cmd := exec.Command(exe)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		hideWindow(cmd)

		if err := cmd.Run(); err != nil {
			log.Printf("watchdog: process exited: %v, restarting in %s", err, restartDelay)
		} else {
			log.Printf("watchdog: process exited cleanly, restarting in %s", restartDelay)
		}
		time.Sleep(restartDelay)
	}
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}