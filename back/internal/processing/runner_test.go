package processing

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/media"
)

type memoryStorage struct{ objects map[string][]byte }

func newMemoryStorage() *memoryStorage { return &memoryStorage{objects: map[string][]byte{}} }
func (s *memoryStorage) Put(_ context.Context, key string, body io.Reader, _ string) error {
	b, err := io.ReadAll(body)
	if err == nil {
		s.objects[key] = b
	}
	return err
}
func (s *memoryStorage) Get(_ context.Context, key string) (media.Object, error) {
	b, ok := s.objects[key]
	if !ok {
		return media.Object{}, os.ErrNotExist
	}
	return media.Object{Key: key, Size: int64(len(b)), Body: io.NopCloser(bytes.NewReader(b))}, nil
}
func (s *memoryStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}
func (s *memoryStorage) Exists(_ context.Context, key string) (bool, error) {
	_, ok := s.objects[key]
	return ok, nil
}
func (s *memoryStorage) PresignGet(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

type fakeDownloader struct{ calls int }

func (d *fakeDownloader) Download(_ context.Context, _ string) (io.ReadCloser, string, error) {
	d.calls++
	return io.NopCloser(bytes.NewReader([]byte("source"))), "video/mp4", nil
}

type fakeMedia struct{ audioCalls, renderCalls int }

func (m *fakeMedia) ExtractAudio(_ context.Context, _, out string) error {
	m.audioCalls++
	return os.WriteFile(out, []byte("audio"), 0o600)
}
func (m *fakeMedia) Render(_ context.Context, in RenderInput) error {
	m.renderCalls++
	return os.WriteFile(in.OutputPath, []byte("render"), 0o600)
}

type fakeTranscriber struct{ calls int }

func (t *fakeTranscriber) Transcribe(_ context.Context, _, outputBase string) error {
	t.calls++
	return os.WriteFile(outputBase+".srt", []byte("1\n00:00:00,000 --> 00:00:01,000\ntext\n"), 0o600)
}

func TestProcessSkipsExistingArtifactsOnRetry(t *testing.T) {
	store, downloader, processor, transcriber := newMemoryStorage(), &fakeDownloader{}, &fakeMedia{}, &fakeTranscriber{}
	runner := Runner{Storage: store, Downloader: downloader, Media: processor, Transcriber: transcriber}
	in := Input{ClipID: "clip-1", ClipURL: "https://example.test/clip", Width: 1080, Height: 1920, Blur: 25, Preset: "veryfast"}

	first, err := runner.Process(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.objects[first.RenderKey]; !ok {
		t.Fatalf("render %q was not stored", first.RenderKey)
	}
	if downloader.calls != 1 || processor.audioCalls != 1 || transcriber.calls != 1 || processor.renderCalls != 1 {
		t.Fatalf("unexpected first-run calls: download=%d audio=%d transcribe=%d render=%d", downloader.calls, processor.audioCalls, transcriber.calls, processor.renderCalls)
	}

	second, err := runner.Process(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("artifact keys changed between retries: %#v != %#v", first, second)
	}
	if downloader.calls != 1 || processor.audioCalls != 1 || transcriber.calls != 1 || processor.renderCalls != 1 {
		t.Fatalf("retry repeated completed work: download=%d audio=%d transcribe=%d render=%d", downloader.calls, processor.audioCalls, transcriber.calls, processor.renderCalls)
	}

}
