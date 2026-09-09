package processing

import (
	"context"
	"io"
	"time"
)

type Downloader interface {
	Download(context.Context, string) (io.ReadCloser, string, error)
}
type MediaProcessor interface {
	ExtractAudio(context.Context, string, string) error
	Render(context.Context, RenderInput) error
}
type Transcriber interface {
	Transcribe(context.Context, string, string) error
}
type RenderInput struct {
	SourcePath, SubtitlePath, OutputPath string
	Width, Height, Blur                  int
	Preset                               string
}
type Job struct {
	ID, ClipID, Type string
	Attempts         int
	CreatedAt        time.Time
}
