package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/finde-clip/finde-v2/back/internal/processing"
)

func TestRenderCreatesVerticalVideoWithBurnedSubtitles(t *testing.T) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	filters, err := exec.Command(bin, "-hide_banner", "-filters").CombinedOutput()
	if err != nil {
		t.Fatalf("inspect ffmpeg filters: %v: %s", err, filters)
	}
	if !strings.Contains(string(filters), "subtitles") {
		t.Skip("ffmpeg was built without the subtitles/libass filter")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	subtitles := filepath.Join(dir, "subtitles.srt")
	output := filepath.Join(dir, "with-subtitles.mp4")

	makeSource := exec.Command(
		bin,
		"-y",
		"-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10",
		"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=44100",
		"-t", "1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-shortest",
		source,
	)
	if log, err := makeSource.CombinedOutput(); err != nil {
		t.Fatalf("create source video: %v: %s", err, log)
	}
	if err := os.WriteFile(subtitles, []byte("1\n00:00:00,000 --> 00:00:00,900\nTest subtitle\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = New(bin).Render(context.Background(), processing.RenderInput{
		SourcePath:   source,
		SubtitlePath: subtitles,
		OutputPath:   output,
		Width:        1080,
		Height:       1920,
		Blur:         10,
		Preset:       "ultrafast",
	})
	if err != nil {
		t.Fatalf("render video: %v", err)
	}

	probe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not installed")
	}
	check := exec.Command(probe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", output)
	got, err := check.Output()
	if err != nil {
		t.Fatalf("inspect rendered video: %v", err)
	}
	if strings.TrimSpace(string(got)) != "1080,1920" {
		t.Fatalf("unexpected rendered dimensions %q", strings.TrimSpace(string(got)))
	}
}

func TestEscapeFilterPath(t *testing.T) {
	got := escapeFilterPath("/tmp/it's: a\\file.srt")
	if got != "/tmp/it\\'s\\: a\\\\file.srt" {
		t.Fatalf("unexpected escaped path %q", got)
	}
}
