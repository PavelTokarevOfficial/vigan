package ffmpeg

import (
	"context"
	"fmt"
	"github.com/finde-clip/finde-v2/back/internal/composition"
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
	config := in.Composition
	if config.Version == 0 {
		config = composition.Default(in.Width, in.Height, in.Blur)
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid render composition: %w", err)
	}

	args := []string{"-y", "-i", in.SourcePath}
	assetInputs, err := appendAssetInputs(&args, config, in.AssetPaths)
	if err != nil {
		return err
	}
	filter, videoLabel, audioLabel, err := buildFilter(config, in.SubtitlePath, assetInputs)
	if err != nil {
		return err
	}
	args = append(args, "-filter_complex", filter, "-map", videoLabel)
	if audioLabel != "" {
		args = append(args, "-map", audioLabel)
	} else {
		args = append(args, "-map", "0:a?")
	}
	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-preset", in.Preset, "-crf", "20", "-c:a", "aac", "-shortest", "-movflags", "+faststart", in.OutputPath)
	return a.run(ctx, args...)
}

func appendAssetInputs(args *[]string, config composition.Config, paths map[string]string) (map[string]int, error) {
	indices := map[string]int{}
	nextIndex := 1 // source Twitch clip is input 0
	for _, layer := range config.Layers {
		if !layer.Visible || layer.AssetID == "" || indices[layer.AssetID] != 0 {
			continue
		}
		path := paths[layer.AssetID]
		if path == "" {
			return nil, fmt.Errorf("render asset %s is missing from job snapshot", layer.AssetID)
		}
		if layer.Type == "image" || layer.Type == "gif" {
			*args = append(*args, "-stream_loop", "-1")
		}
		*args = append(*args, "-i", path)
		indices[layer.AssetID] = nextIndex
		nextIndex++
	}
	return indices, nil
}

func buildFilter(config composition.Config, subtitlePath string, assetInputs map[string]int) (string, string, string, error) {
	filters := []string{fmt.Sprintf("color=c=%s:s=%dx%d:r=%d[base0]", safeColor(config.Canvas.Background, "#000000"), config.Canvas.Width, config.Canvas.Height, config.Canvas.FPS)}
	base := "base0"
	step := 0
	audioLabels := []string{}
	for _, layer := range config.Layers {
		if !layer.Visible {
			continue
		}
		if layer.Type == "audio" {
			index, ok := assetInputs[layer.AssetID]
			if !ok {
				return "", "", "", fmt.Errorf("audio layer %s has no resolved asset", layer.ID)
			}
			label := fmt.Sprintf("audio%d", len(audioLabels))
			filters = append(filters, fmt.Sprintf("[%d:a]volume=1[%s]", index, label))
			audioLabels = append(audioLabels, "["+label+"]")
			continue
		}
		next := fmt.Sprintf("base%d", step+1)
		switch layer.Type {
		case "subtitles":
			style := layer.Style
			fontSize := positiveOr(style.FontSize, 8)
			alignment := positiveOr(style.Alignment, 2)
			marginV := style.MarginV
			if marginV <= 0 {
				marginV = max(0, config.Canvas.Height-(layer.Y+layer.Height))
			}
			outline := nonNegativeOr(style.Outline, 2)
			primary := safeASSColor(style.PrimaryColor, "&H00FFFFFF")
			outlineColor := safeASSColor(style.OutlineColor, "&H00000000")
			// This is the point where Whisper's local SRT is burned into the render.
			filters = append(filters, fmt.Sprintf("[%s]subtitles=filename='%s':force_style='Alignment=%d,MarginV=%d,Fontsize=%d,PrimaryColour=%s,OutlineColour=%s,BorderStyle=1,Outline=%d'[%s]", base, escapeFilterPath(subtitlePath), alignment, marginV, fontSize, primary, outlineColor, outline, next))
		case "text":
			fontSize := positiveOr(layer.Style.FontSize, max(18, layer.Height/5))
			filters = append(filters, fmt.Sprintf("[%s]drawtext=text='%s':x=%d:y=%d:fontsize=%d:fontcolor=white:borderw=%d:bordercolor=black[%s]", base, escapeDrawText(layer.Text), layer.X, layer.Y, fontSize, nonNegativeOr(layer.Style.Outline, 2), next))
		case "color":
			visual := fmt.Sprintf("layer%d", step)
			filters = append(filters, fmt.Sprintf("color=c=%s:s=%dx%d:r=%d[%s]", safeColor(layer.Color, "#000000"), videoDimension(layer.Width), videoDimension(layer.Height), config.Canvas.FPS, visual))
			filters = append(filters, overlayFilter(base, visual, next, layer))
		default:
			input := "[0:v]"
			if layer.Type != "input_video" {
				index, ok := assetInputs[layer.AssetID]
				if !ok {
					return "", "", "", fmt.Errorf("layer %s has no resolved asset", layer.ID)
				}
				input = fmt.Sprintf("[%d:v]", index)
			}
			visual := fmt.Sprintf("layer%d", step)
			filters = append(filters, input+scaleFilter(layer)+"["+visual+"]")
			filters = append(filters, overlayFilter(base, visual, next, layer))
		}
		base = next
		step++
	}
	audio := ""
	if len(audioLabels) > 0 {
		audio = "[audioout]"
		filters = append(filters, "[0:a]"+strings.Join(audioLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=first:dropout_transition=0[audioout]", len(audioLabels)+1))
	}
	return strings.Join(filters, ";"), "[" + base + "]", audio, nil
}

func scaleFilter(layer composition.Layer) string {
	width, height := videoDimension(layer.Width), videoDimension(layer.Height)
	fit := layer.Fit
	if fit == "" {
		fit = "contain"
	}
	base := "scale="
	switch fit {
	case "cover":
		base += fmt.Sprintf("%d:%d:force_original_aspect_ratio=increase:force_divisible_by=2,crop=%d:%d", width, height, width, height)
	case "stretch":
		base += fmt.Sprintf("%d:%d", width, height)
	default:
		base += fmt.Sprintf("%d:%d:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black", width, height, width, height)
	}
	if layer.Filters.Blur > 0 {
		base += fmt.Sprintf(",boxblur=%d:10", layer.Filters.Blur)
	}
	if layer.Filters.Brightness != 0 {
		base += fmt.Sprintf(",eq=brightness=%g", layer.Filters.Brightness)
	}
	if layer.Opacity < 1 {
		base += fmt.Sprintf(",format=rgba,colorchannelmixer=aa=%g", layer.Opacity)
	}
	return base
}

// H.264 and the scale/pad chain are most reliable with even intermediate
// dimensions. The editor may contain an odd user-entered value (e.g. 607);
// rendering it as 606 avoids FFmpeg rounding it up past the pad target.
func videoDimension(value int) int {
	if value < 2 {
		return 2
	}
	if value%2 != 0 {
		return value - 1
	}
	return value
}

func overlayFilter(base, visual, next string, layer composition.Layer) string {
	return fmt.Sprintf("[%s][%s]overlay=%d:%d:shortest=1[%s]", base, visual, layer.X, layer.Y, next)
}

func positiveOr(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
func nonNegativeOr(value, fallback int) int {
	if value >= 0 {
		return value
	}
	return fallback
}
func safeColor(value, fallback string) string {
	if strings.HasPrefix(value, "#") && len(value) == 7 {
		return value
	}
	return fallback
}
func safeASSColor(value, fallback string) string {
	if strings.HasPrefix(value, "&H") && len(value) == 10 {
		return value
	}
	return fallback
}
func escapeDrawText(value string) string {
	return strings.NewReplacer("\\", "\\\\", "'", "\\'", ":", "\\:", "\n", "\\n").Replace(value)
}

func escapeFilterPath(path string) string {
	return strings.NewReplacer("\\", "\\\\", ":", "\\:", "'", "\\'").Replace(path)
}
