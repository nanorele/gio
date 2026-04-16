package debug

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	debugVariable = "GIODEBUG"
	textSubsystem = "text"
	silentFeature = "silent"
)

var Text atomic.Bool

var parseOnce sync.Once

func Parse() {
	parseOnce.Do(func() {
		val, ok := os.LookupEnv(debugVariable)
		if !ok {
			return
		}
		print := false
		silent := false
		for part := range strings.SplitSeq(val, ",") {
			switch part {
			case textSubsystem:
				Text.Store(true)
			case silentFeature:
				silent = true
			default:
				print = true
			}
		}
		if print && !silent {
			fmt.Fprintf(os.Stderr,
				`Usage of %s:
	A comma-delimited list of debug subsystems to enable. Currently recognized systems:

	- %s: text debug info including system font resolution
	- %s: silence this usage message even if GIODEBUG contains invalid content
`, debugVariable, textSubsystem, silentFeature)
		}
	})
}
