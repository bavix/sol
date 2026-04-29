package system

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/bavix/sol/internal/domain/wol"
)

var ErrUnsupportedOS = errors.New("unsupported operating system")

type PowerController struct{}

func NewPowerController() *PowerController {
	return &PowerController{}
}

func (p *PowerController) Execute(ctx context.Context, action wol.Action) error {
	switch action {
	case wol.ActionShutdown:
		return p.shutdown(ctx)
	case wol.ActionReboot:
		return p.reboot(ctx)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, action)
	}
}

func (p *PowerController) shutdown(ctx context.Context) error {
	switch runtime.GOOS {
	case "windows":
		return execCmd(ctx, "shutdown", "-s", "-t", "0", "-f")
	case "linux", "darwin":
		return execCmd(ctx, "shutdown", "-h", "now")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
}

func (p *PowerController) reboot(ctx context.Context) error {
	switch runtime.GOOS {
	case "windows":
		return execCmd(ctx, "shutdown", "-r", "-t", "0", "-f")
	case "linux", "darwin":
		return execCmd(ctx, "shutdown", "-r", "now")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
}

func execCmd(ctx context.Context, name string, args ...string) error { //nolint:unparam
	// name is always "shutdown" in practice, but the function is generic by design
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
