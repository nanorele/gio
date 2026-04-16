package key

import (
	"strings"
	"syscall/js"
)

var ModShortcut = ModCtrl

var ModShortcutAlt = ModCtrl

func init() {
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return
	}

	platform := ""
	if p := nav.Get("platform"); p.Truthy() {
		platform = p.String()
	}
	platform = strings.ToLower(platform)

	for _, darwinPlatform := range []string{"mac", "iphone", "ipad", "ipod"} {
		if strings.HasPrefix(platform, darwinPlatform) {
			ModShortcut = ModCommand
			ModShortcutAlt = ModAlt
			return
		}
	}
}
