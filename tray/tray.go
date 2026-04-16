//go:build windows

package tray

import (
	"os/exec"

	"github.com/getlantern/systray"
)

func Run(url string) {
	systray.Run(func() { onReady(url) }, func() {})
}

func onReady(url string) {
	systray.SetIcon(icon)
	systray.SetTitle("Tibiantis Account Switcher")
	systray.SetTooltip("Tibiantis Account Switcher")

	mOpen := systray.AddMenuItem("Open", "Open in browser")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser(url)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func openBrowser(url string) {
	exec.Command("explorer", url).Start()
}
