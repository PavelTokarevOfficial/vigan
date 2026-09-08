package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr, DatabaseURL, S3Endpoint, S3PublicEndpoint, S3AccessKey, S3SecretKey, S3Bucket, S3Region, TwitchClientID, TwitchClientSecret, FFMPEG, Whisper, WhisperModel, BrowserBin string
	FFmpegPreset                                                                                                                                                                     string
	OutputWidth, OutputHeight, BackgroundBlur, BannerTop                                                                                                                             int
	BannerScale                                                                                                                                                                      float64
	S3SSL, BrowserHeadless                                                                                                                                                           bool
}

func Load() (Config, error) {
	// Docker Compose injects env itself. Local `go run` loads the root .env without overriding exported values.
	_ = godotenv.Load("../.env", ".env")
	c := Config{HTTPAddr: get("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), S3Endpoint: os.Getenv("S3_ENDPOINT"), S3PublicEndpoint: get("S3_PUBLIC_ENDPOINT", "http://localhost:9000"), S3AccessKey: os.Getenv("S3_ACCESS_KEY"), S3SecretKey: os.Getenv("S3_SECRET_KEY"), S3Bucket: os.Getenv("S3_BUCKET"), S3Region: get("S3_REGION", "us-east-1"), TwitchClientID: os.Getenv("TWITCH_CLIENT_ID"), TwitchClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"), FFMPEG: get("FFMPEG_BIN_PATH", "ffmpeg"), Whisper: get("WHISPER_BIN_PATH", "whisper-cli"), WhisperModel: os.Getenv("WHISPER_MODEL_PATH"), BrowserBin: os.Getenv("BROWSER_BIN_PATH")}
	c.S3SSL, _ = strconv.ParseBool(get("S3_USE_SSL", "false"))
	if !runningInContainer() {
		// Compose service DNS names resolve only inside Docker; local `go run` uses published ports.
		c.DatabaseURL = localServiceURL(c.DatabaseURL, "postgres", "5432")
		c.S3Endpoint = localServiceURL(c.S3Endpoint, "minio", "9000")
	}
	c.BrowserHeadless, _ = strconv.ParseBool(get("BROWSER_HEADLESS", "true"))
	c.FFmpegPreset = get("FFMPEG_PRESET", "veryfast")
	var err error
	if c.OutputWidth, err = getInt("OUTPUT_WIDTH", 1080); err != nil {
		return c, err
	}
	if c.OutputHeight, err = getInt("OUTPUT_HEIGHT", 1920); err != nil {
		return c, err
	}
	if c.BackgroundBlur, err = getInt("BACKGROUND_BLUR", 25); err != nil {
		return c, err
	}
	if c.BannerTop, err = getInt("BANNER_TOP", 100); err != nil {
		return c, err
	}
	if c.BannerScale, err = getFloat("BANNER_SCALE", .8); err != nil {
		return c, err
	}
	for k, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "S3_ENDPOINT": c.S3Endpoint, "S3_ACCESS_KEY": c.S3AccessKey, "S3_SECRET_KEY": c.S3SecretKey, "S3_BUCKET": c.S3Bucket, "TWITCH_CLIENT_ID": c.TwitchClientID, "TWITCH_CLIENT_SECRET": c.TwitchClientSecret} {
		if v == "" {
			return c, fmt.Errorf("required config %s is empty", k)
		}
	}
	return c, nil
}

func runningInContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func localServiceURL(raw, service, fallbackPort string) string {
	plain := strings.TrimPrefix(strings.TrimPrefix(raw, "http://"), "https://")
	if plain == service || plain == service+":"+fallbackPort {
		return strings.Replace(raw, service, "localhost", 1)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() != service {
		return raw
	}
	port := u.Port()
	if port == "" {
		port = fallbackPort
	}
	u.Host = net.JoinHostPort("localhost", port)
	return u.String()
}

// ValidateWorker keeps API usable before a local Whisper model is installed.
func (c Config) ValidateWorker() error {
	if c.WhisperModel == "" {
		return fmt.Errorf("required config WHISPER_MODEL_PATH is empty")
	}
	return nil
}

func getInt(key string, fallback int) (int, error) {
	v := get(key, strconv.Itoa(fallback))
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return n, nil
}

func getFloat(key string, fallback float64) (float64, error) {
	v := get(key, strconv.FormatFloat(fallback, 'f', -1, 64))
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", key)
	}
	return n, nil
}
func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
