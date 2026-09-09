package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/finde-clip/finde-v2/back/infrastructure/browser"
	"github.com/finde-clip/finde-v2/back/infrastructure/ffmpeg"
	"github.com/finde-clip/finde-v2/back/infrastructure/storage"
	"github.com/finde-clip/finde-v2/back/infrastructure/whisper"
	"github.com/finde-clip/finde-v2/back/internal/config"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/finde-clip/finde-v2/back/internal/platform/db"
	"github.com/finde-clip/finde-v2/back/internal/processing"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, e := config.Load()
	if e != nil {
		log.Error("invalid config", "error", e)
		os.Exit(1)
	}
	if e = cfg.ValidateWorker(); e != nil {
		log.Error("invalid worker config", "error", e)
		os.Exit(1)
	}
	if e = db.Migrate(cfg.DatabaseURL, "migrations"); e != nil {
		log.Error("migration failed", "error", e)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	pool, e := db.Open(ctx, cfg.DatabaseURL)
	if e != nil {
		log.Error("database unavailable", "error", e)
		os.Exit(1)
	}
	defer pool.Close()
	store, e := storage.New(ctx, storage.Settings{Endpoint: cfg.S3Endpoint, PublicEndpoint: cfg.S3PublicEndpoint, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, Bucket: cfg.S3Bucket, Region: cfg.S3Region, UseSSL: cfg.S3SSL})
	if e != nil {
		log.Error("storage unavailable", "error", e)
		os.Exit(1)
	}
	if e = store.EnsureBucket(ctx); e != nil {
		log.Error("bucket unavailable", "error", e)
		os.Exit(1)
	}
	jobs := processing.NewJobs(pool)
	files := media.NewFiles(pool)
	down := browser.NewRodDownloader(cfg.BrowserHeadless, cfg.BrowserBin)
	runner := &processing.Runner{Storage: store, Downloader: down, Media: ffmpeg.New(cfg.FFMPEG), Transcriber: whisper.New(cfg.Whisper, cfg.WhisperModel)}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	log.Info("worker started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			job, e := jobs.Claim(ctx)
			if e != nil {
				if !errors.Is(e, pgx.ErrNoRows) && !errors.Is(e, context.Canceled) {
					log.Error("job claim failed", "error", e)
				}
				continue
			}
			started := time.Now()
			log.Info("job started", "job_id", job.ID, "clip_id", job.ClipID, "type", job.Type)
			e = run(ctx, log, jobs, files, runner, store, down, cfg, job)
			if e != nil {
				if errors.Is(e, context.Canceled) {
					shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
					_ = jobs.Requeue(shutdownCtx, job.ID)
					shutdownCancel()
					log.Warn("job requeued during shutdown", "job_id", job.ID, "clip_id", job.ClipID)
					return
				}
				_ = jobs.Fail(ctx, job.ID, job.ClipID, e.Error())
				log.Error("job failed", "job_id", job.ID, "error", e)
			} else {
				clipStatus := "completed"
				if job.Type == "download" {
					clipStatus = "downloaded"
				}
				_ = jobs.Complete(ctx, job.ID, job.ClipID, clipStatus)
				log.Info("job completed", "job_id", job.ID, "duration", time.Since(started).String())
			}
		}
	}
}
func run(ctx context.Context, log *slog.Logger, j *processing.Jobs, files *media.Files, r *processing.Runner, s *storage.S3, d *browser.RodDownloader, cfg config.Config, job processing.ClaimedJob) error {
	if job.Type == "download" {
		log.Info("job step", "job_id", job.ID, "clip_id", job.ClipID, "step", "download", "progress", 10)
		if e := j.Step(ctx, job.ID, job.ClipID, "download", "downloading", 10); e != nil {
			return e
		}
		body, mime, e := d.Download(ctx, job.ClipURL)
		if e != nil {
			return e
		}
		defer body.Close()
		if mime == "" {
			mime = "video/mp4"
		}
		if e = s.Put(ctx, "sources/"+job.ClipID+"/source.mp4", body, mime); e != nil {
			return e
		}
		if e = record(ctx, files, s, job.ClipID, job.ID, "source", "sources/"+job.ClipID+"/source.mp4", mime); e != nil {
			return e
		}
		return j.Step(ctx, job.ID, job.ClipID, "downloaded", "downloaded", 100)
	}
	if job.Type != "process" {
		return fmt.Errorf("unknown job type %s", job.Type)
	}
	if e := j.Step(ctx, job.ID, job.ClipID, "processing_queued", "downloaded", 20); e != nil {
		return e
	}
	log.Info("job step", "job_id", job.ID, "clip_id", job.ClipID, "step", "processing_queued", "progress", 20)
	out, e := r.Process(ctx, processing.Input{JobID: job.ID, ClipID: job.ClipID, ClipURL: job.ClipURL, Width: cfg.OutputWidth, Height: cfg.OutputHeight, Blur: cfg.BackgroundBlur, Preset: cfg.FFmpegPreset, TemplateSnapshot: job.TemplateSnapshot, Progress: func(step, status string, percent int) error {
		log.Info("job step", "job_id", job.ID, "clip_id", job.ClipID, "step", step, "progress", percent)
		return j.Step(ctx, job.ID, job.ClipID, step, status, percent)
	}})
	if e != nil {
		return e
	}
	for _, x := range []struct{ kind, key, mime string }{{"audio", out.AudioKey, "audio/wav"}, {"subtitle", out.SubtitleKey, "application/x-subrip"}, {"render", out.RenderKey, "video/mp4"}} {
		if e = record(ctx, files, s, job.ClipID, job.ID, x.kind, x.key, x.mime); e != nil {
			return e
		}
	}
	log.Info("job step", "job_id", job.ID, "clip_id", job.ClipID, "step", "rendered", "progress", 90)
	return j.Step(ctx, job.ID, job.ClipID, "rendered", "rendering", 90)
}
func record(ctx context.Context, f *media.Files, s *storage.S3, clipID, jobID, kind, key, mime string) error {
	o, e := s.Get(ctx, key)
	if e != nil {
		return e
	}
	defer o.Body.Close()
	return f.Upsert(ctx, clipID, jobID, kind, key, mime, o.Size)
}
