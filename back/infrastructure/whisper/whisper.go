package whisper

import (
	"context"
	"fmt"
	"os/exec"
)

type Adapter struct{ Bin, Model string }

func New(bin, model string) *Adapter { return &Adapter{bin, model} }
func (a *Adapter) Transcribe(ctx context.Context, audio, outputBase string) error {
	out, e := exec.CommandContext(ctx, a.Bin, "-m", a.Model, "-l", "auto", "-osrt", "-ml", "10", "-sow", "-of", outputBase, "-f", audio).CombinedOutput()
	if e != nil {
		return fmt.Errorf("whisper: %w: %s", e, string(out))
	}
	return nil
}
