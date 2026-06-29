//go:build windows

package tray

import (
	_ "embed"
	"os/exec"

	"github.com/getlantern/systray"
)

//go:embed favicon.ico
var favicon []byte

func Run(url string, onQuit func()) {
	systray.Run(func() { onReady(url) }, onQuit)
}

func onReady(url string) {
	systray.SetIcon(favicon)
	systray.SetTooltip("Tibiantis Assistant")

	mOpen := systray.AddMenuItem("Open", "Open in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")

	go func() {
		for {
			select {
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
