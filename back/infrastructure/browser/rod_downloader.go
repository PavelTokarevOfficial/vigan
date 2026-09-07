package browser

import (
	"context"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"io"
	"net/http"
	"time"
)

// RodDownloader owns all Twitch-page selectors and can be replaced without touching pipeline code.
type RodDownloader struct {
	headless bool
	bin      string
	client   *http.Client
}

func NewRodDownloader(headless bool, bin string) *RodDownloader {
	return &RodDownloader{headless: headless, bin: bin, client: &http.Client{Timeout: 2 * time.Minute}}
}
func (d *RodDownloader) Download(ctx context.Context, clipURL string) (io.ReadCloser, string, error) {
	l := launcher.New().Headless(d.headless)
	if d.bin != "" {
		l.Bin(d.bin)
	}
	u, e := l.Launch()
	if e != nil {
		return nil, "", fmt.Errorf("launch Chromium: %w", e)
	}
	b := rod.New().ControlURL(u)
	if e = b.Connect(); e != nil {
		return nil, "", fmt.Errorf("connect Chromium: %w", e)
	}
	defer func() { _ = b.Close() }()
	p, e := b.Context(ctx).Page(proto.TargetCreateTarget{})
	if e != nil {
		return nil, "", e
	}
	if e = p.Navigate(clipURL); e != nil {
		return nil, "", e
	}
	if e = p.WaitLoad(); e != nil {
		return nil, "", e
	}
	// Keep player extraction here: Twitch may attach the MP4 after the <video> element appears.
	e = p.Timeout(20*time.Second).WaitElementsMoreThan("video", 0)
	if e != nil {
		return nil, "", fmt.Errorf("twitch player video not found: %w", e)
	}
	v, e := p.Timeout(20 * time.Second).Eval(`async () => {
		const directMP4 = () => {
			const video = [...document.querySelectorAll('video')]
				.map((v) => v.currentSrc || v.src || v.querySelector('source')?.src)
				.find((src) => /^https?:\/\/.+\.mp4(?:[?#].*)?$/i.test(src || ''));
			if (video) return video;
			return performance.getEntriesByType('resource')
				.map((entry) => entry.name)
				.find((src) => /^https?:\/\/.+\.mp4(?:[?#].*)?$/i.test(src));
		};
		const until = Date.now() + 19000;
		while (Date.now() < until) {
			const src = directMP4();
			if (src) return src;
			await new Promise((resolve) => setTimeout(resolve, 500));
		}
		return '';
	}`)
	if e != nil {
		return nil, "", e
	}
	src := v.Value.Str()
	if src == "" {
		return nil, "", fmt.Errorf("twitch player did not expose a direct MP4 source")
	}
	ua, e := p.Eval(`() => navigator.userAgent`)
	if e != nil {
		return nil, "", e
	}
	r, e := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if e != nil {
		return nil, "", e
	}
	r.Header.Set("Referer", clipURL)
	r.Header.Set("User-Agent", ua.Value.Str())
	res, e := d.client.Do(r)
	if e != nil {
		return nil, "", e
	}
	if res.StatusCode/100 != 2 {
		res.Body.Close()
		return nil, "", fmt.Errorf("clip media status %d", res.StatusCode)
	}
	return res.Body, res.Header.Get("Content-Type"), nil
}
