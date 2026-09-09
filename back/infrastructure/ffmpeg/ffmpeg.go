package ffmpeg

import (
	"context"
	"fmt"
	"github.com/finde-clip/finde-v2/back/internal/processing"
	"os/exec"
	"strings"
)

type Adapter struct{ Bin string }

func New(bin string) *Adapter { return &Adapter{bin} }
func (a *Adapter) run(ctx context.Context, args ...string) error {
	out, e := exec.CommandContext(ctx, a.Bin, args...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("ffmpeg: %w: %s", e, string(out))
	}
	return nil
}
func (a *Adapter) ExtractAudio(ctx context.Context, source, out string) error {
	return a.run(ctx, "-y", "-i", source, "-ar", "16000", "-ac", "1", out)
}
func (a *Adapter) Render(ctx context.Context, in processing.RenderInput) error {
	filter := fmt.Sprintf("[0:v]split=2[bg][fg];[bg]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=%d:10,eq=brightness=-0.2[bg];[fg]scale=%d:-2:force_original_aspect_ratio=decrease[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2[base]", in.Width, in.Height, in.Width, in.Height, in.Blur, in.Width)
	args := []string{"-y", "-i", in.SourcePath}
	// The subtitle filter consumes the local SRT created by Whisper; the result is burned into the MP4.
	filter += fmt.Sprintf(";[base]subtitles=filename='%s':force_style='Alignment=2,MarginV=100,Fontsize=8,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,BorderStyle=1,Outline=2'[out]", escapeFilterPath(in.SubtitlePath))
	args = append(args, "-filter_complex", filter, "-map", "[out]", "-map", "0:a?", "-c:v", "libx264", "-preset", in.Preset, "-crf", "20", "-c:a", "aac", "-movflags", "+faststart", in.OutputPath)
	return a.run(ctx, args...)
}

func escapeFilterPath(path string) string {
	return strings.NewReplacer("\\", "\\\\", ":", "\\:", "'", "\\'").Replace(path)
}
