//go:build windows

package tray

import (
	"context"
	_ "embed"
	"os/exec"

	"github.com/getlantern/systray"
)

//go:embed favicon.ico
var favicon []byte

func Run(ctx context.Context, url string, onQuit func()) {
	systray.Run(func() { onReady(ctx, url) }, onQuit)
}

func onReady(ctx context.Context, url string) {
	systray.SetIcon(favicon)
	systray.SetTooltip("Tibiantis Assistant")

	mOpen := systray.AddMenuItem("Open", "Open in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")

	go func() {
		for {
			select {
			case <-ctx.Done():
				systray.Quit()
				return
			case <-mOpen.ClickedCh:
				OpenBrowser(url)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func OpenBrowser(url string) {
	_ = exec.Command("explorer", url).Start()
}
