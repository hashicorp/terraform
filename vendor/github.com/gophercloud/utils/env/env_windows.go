package env

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/charmap"
)

func Getenv(s string) string {
	var st uint32
	env := os.Getenv(s)
	if windows.GetConsoleMode(windows.Handle(syscall.Stdin), &st) == nil ||
		windows.GetConsoleMode(windows.Handle(syscall.Stdout), &st) == nil ||
		windows.GetConsoleMode(windows.Handle(syscall.Stderr), &st) == nil {
		// detect windows console, should be skipped in cygwin environment
		var cm charmap.Charmap
		switch windows.GetACP() {
		case 37:
			cm = *charmap.CodePage037
		case 1047:
			cm = *charmap.CodePage1047
		case 1140:
			cm = *charmap.CodePage1140
		case 437:
			cm = *charmap.CodePage437
		case 850:
			cm = *charmap.CodePage850
		case 852:
			cm = *charmap.CodePage852
		case 855:
			cm = *charmap.CodePage855
		case 858:
			cm = *charmap.CodePage858
		case 860:
			cm = *charmap.CodePage860
		case 862:
			cm = *charmap.CodePage862
		case 863:
			cm = *charmap.CodePage863
		case 865:
			cm = *charmap.CodePage865
		case 866:
			cm = *charmap.CodePage866
		case 28591:
			cm = *charmap.ISO8859_1
		case 28592:
			cm = *charmap.ISO8859_2
		case 28593:
			cm = *charmap.ISO8859_3
		case 28594:
			cm = *charmap.ISO8859_4
		case 28595:
			cm = *charmap.ISO8859_5
		case 28596:
			cm = *charmap.ISO8859_6
		case 28597:
			cm = *charmap.ISO8859_7
		case 28598:
			cm = *charmap.ISO8859_8
		case 28599:
			cm = *charmap.ISO8859_9
		case 28600:
			cm = *charmap.ISO8859_10
		case 28603:
			cm = *charmap.ISO8859_13
		case 28604:
			cm = *charmap.ISO8859_14
		case 28605:
			cm = *charmap.ISO8859_15
		case 28606:
			cm = *charmap.ISO8859_16
		case 20866:
			cm = *charmap.KOI8R
		case 21866:
			cm = *charmap.KOI8U
		case 1250:
			cm = *charmap.Windows1250
		case 1251:
			cm = *charmap.Windows1251
		case 1252:
			cm = *charmap.Windows1252
		case 1253:
			cm = *charmap.Windows1253
		case 1254:
			cm = *charmap.Windows1254
		case 1255:
			cm = *charmap.Windows1255
		case 1256:
			cm = *charmap.Windows1256
		case 1257:
			cm = *charmap.Windows1257
		case 1258:
			cm = *charmap.Windows1258
		case 874:
			cm = *charmap.Windows874
		default:
			return env
		}
		if v, err := cm.NewEncoder().String(env); err == nil {
			return v
		}
	}
	return env
}
