package processing

import (
	"context"
	"fmt"
	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"io"
	"os"
	"path/filepath"
)

// Runner orchestrates steps; it does not know Twitch, Rod, S3 or CLI details.
type Runner struct {
	Storage     media.Storage
	Downloader  Downloader
	Media       MediaProcessor
	Transcriber Transcriber
}
type Input struct {
	JobID, ClipID, ClipURL string
	Width, Height, Blur    int
	Preset                 string
	TemplateSnapshot       []byte
	Progress               func(step, clipStatus string, percent int) error
}
type Result struct{ SourceKey, AudioKey, SubtitleKey, RenderKey string }

func (r *Runner) Process(ctx context.Context, in Input) (Result, error) {
	d, e := os.MkdirTemp("", "finde-job-")
	if e != nil {
		return Result{}, e
	}
	defer os.RemoveAll(d)
	renderName := in.JobID
	if renderName == "" {
		renderName = "vertical"
	}
	out := Result{SourceKey: "sources/" + in.ClipID + "/source.mp4", AudioKey: "audio/" + in.ClipID + "/audio.wav", SubtitleKey: "subtitles/" + in.ClipID + "/subtitles.srt", RenderKey: "renders/" + in.ClipID + "/" + renderName + ".mp4"}
	src := filepath.Join(d, "source.mp4")
	if e = r.ensureSource(ctx, in.ClipURL, out.SourceKey, src); e != nil {
		return out, e
	}
	audio := filepath.Join(d, "audio.wav")
	if ok, e := r.Storage.Exists(ctx, out.AudioKey); e != nil {
		return out, e
	} else if !ok {
		if e = report(in, "extracting_audio", "transcribing", 35); e != nil {
			return out, e
		}
		if e = r.Media.ExtractAudio(ctx, src, audio); e != nil {
			return out, fmt.Errorf("extract audio: %w", e)
		}
		if e = r.putFile(ctx, out.AudioKey, audio, "audio/wav"); e != nil {
			return out, e
		}
	}
	if ok, e := r.Storage.Exists(ctx, out.SubtitleKey); e != nil {
		return out, e
	} else if !ok {
		if e = report(in, "transcribing", "transcribing", 55); e != nil {
			return out, e
		}
		if e = r.ensureLocal(ctx, out.AudioKey, audio); e != nil {
			return out, e
		}
		base := filepath.Join(d, "subtitles")
		if e = r.Transcriber.Transcribe(ctx, audio, base); e != nil {
			return out, fmt.Errorf("transcribe: %w", e)
		}
		if e = r.putFile(ctx, out.SubtitleKey, base+".srt", "application/x-subrip"); e != nil {
			return out, e
		}
	}
	if ok, e := r.Storage.Exists(ctx, out.RenderKey); e != nil {
		return out, e
	} else if !ok {
		if e = report(in, "rendering", "rendering", 80); e != nil {
			return out, e
		}
		sub := filepath.Join(d, "subtitles.srt")
		if e = r.ensureLocal(ctx, out.SubtitleKey, sub); e != nil {
			return out, e
		}
		config, assets, e := r.templateFiles(ctx, in, d)
		if e != nil {
			return out, e
		}
		render := filepath.Join(d, "final.mp4")
		if e = r.Media.Render(ctx, RenderInput{SourcePath: src, SubtitlePath: sub, OutputPath: render, Width: in.Width, Height: in.Height, Blur: in.Blur, Preset: in.Preset, Composition: config, AssetPaths: assets}); e != nil {
			return out, fmt.Errorf("render: %w", e)
		}
		if e = r.putFile(ctx, out.RenderKey, render, "video/mp4"); e != nil {
			return out, e
		}
	}
	return out, nil
}

// templateFiles resolves the immutable asset keys from the job snapshot, never
// from the mutable template record. The FFmpeg adapter only receives local paths.
func (r *Runner) templateFiles(ctx context.Context, in Input, dir string) (composition.Config, map[string]string, error) {
	config := composition.Default(in.Width, in.Height, in.Blur)
	if len(in.TemplateSnapshot) == 0 {
		return config, map[string]string{}, nil
	}
	snapshot, err := composition.ParseSnapshot(in.TemplateSnapshot)
	if err != nil {
		return composition.Config{}, nil, err
	}
	paths := make(map[string]string, len(snapshot.Assets))
	for index, asset := range snapshot.Assets {
		if asset.ID == "" || asset.StorageKey == "" {
			return composition.Config{}, nil, fmt.Errorf("template snapshot contains an invalid asset")
		}
		path := filepath.Join(dir, fmt.Sprintf("asset-%d", index))
		if err := r.ensureLocal(ctx, asset.StorageKey, path); err != nil {
			return composition.Config{}, nil, fmt.Errorf("download template asset %s: %w", asset.ID, err)
		}
		paths[asset.ID] = path
	}
	return snapshot.Config, paths, nil
}

func report(in Input, step, clipStatus string, percent int) error {
	if in.Progress == nil {
		return nil
	}
	return in.Progress(step, clipStatus, percent)
}
func (r *Runner) ensureSource(ctx context.Context, url, key, local string) error {
	ok, e := r.Storage.Exists(ctx, key)
	if e != nil {
		return e
	}
	if ok {
		return r.ensureLocal(ctx, key, local)
	}
	body, mime, e := r.Downloader.Download(ctx, url)
	if e != nil {
		return fmt.Errorf("download: %w", e)
	}
	defer body.Close()
	if mime == "" {
		mime = "video/mp4"
	}
	if e = r.Storage.Put(ctx, key, body, mime); e != nil {
		return e
	}
	return r.ensureLocal(ctx, key, local)
}
func (r *Runner) ensureLocal(ctx context.Context, key, path string) error {
	o, e := r.Storage.Get(ctx, key)
	if e != nil {
		return e
	}
	defer o.Body.Close()
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = io.Copy(f, o.Body)
	return e
}
func (r *Runner) putFile(ctx context.Context, key, path, mime string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return r.Storage.Put(ctx, key, f, mime)
}
