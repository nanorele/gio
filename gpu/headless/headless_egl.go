//go:build linux || freebsd || openbsd

package headless

import (
	"github.com/nanorele/gio/internal/egl"
)

func init() {
	newContextPrimary = func() (context, error) {
		return egl.NewContext(egl.EGL_DEFAULT_DISPLAY)
	}
}
