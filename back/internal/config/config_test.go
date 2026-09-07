package config

import "testing"

func TestValidateWorkerRequiresWhisperModel(t *testing.T) {
	if err := (Config{}).ValidateWorker(); err == nil {
		t.Fatal("empty Whisper model must be rejected for worker")
	}
	if err := (Config{WhisperModel: "/models/ggml-tiny.bin"}).ValidateWorker(); err != nil {
		t.Fatalf("configured Whisper model must be accepted: %v", err)
	}
}

func TestLoadAllowsAPIBeforeWhisperModelIsInstalled(t *testing.T) {
	for key, value := range map[string]string{
		"DATABASE_URL":         "postgres://test",
		"S3_ENDPOINT":          "minio:9000",
		"S3_ACCESS_KEY":        "test",
		"S3_SECRET_KEY":        "test",
		"S3_BUCKET":            "clips",
		"TWITCH_CLIENT_ID":     "test",
		"TWITCH_CLIENT_SECRET": "test",
		"WHISPER_MODEL_PATH":   "",
	} {
		t.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("API config without model: %v", err)
	}
	if err := cfg.ValidateWorker(); err == nil {
		t.Fatal("worker must still require the Whisper model")
	}
}
