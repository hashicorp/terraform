package terminal

import (
	"os"
	"regexp"

	"github.com/fatih/color"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	red            color.Attribute = color.FgRed
	green                          = color.FgGreen
	yellow                         = color.FgYellow
	magenta                        = color.FgMagenta
	cyan                           = color.FgCyan
	grey                           = color.FgWhite
	defaultFgColor                 = 38
)

var (
	colorize               func(message string, textColor color.Attribute, bold int) string
	TerminalSupportsColors = isTerminal()
	UserAskedForColors     = ""
)

func init() {
	InitColorSupport()
}

func InitColorSupport() {
	if colorsEnabled() {
		colorize = func(message string, textColor color.Attribute, bold int) string {
			colorPrinter := color.New(textColor)
			if bold == 1 {
				colorPrinter = colorPrinter.Add(color.Bold)
			}
			f := colorPrinter.SprintFunc()
			return f(message)
		}
	} else {
		colorize = func(message string, _ color.Attribute, _ int) string {
			return message
		}
	}
}

func colorsEnabled() bool {
	if os.Getenv("CF_COLOR") == "true" {
		return true
	}

	if os.Getenv("CF_COLOR") == "false" {
		return false
	}

	if UserAskedForColors == "true" {
		return true
	}

	return UserAskedForColors != "false" && TerminalSupportsColors
}

func Colorize(message string, textColor color.Attribute) string {
	return colorize(message, textColor, 0)
}

func ColorizeBold(message string, textColor color.Attribute) string {
	return colorize(message, textColor, 1)
}

var decolorizerRegex = regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)

func Decolorize(message string) string {
	return string(decolorizerRegex.ReplaceAll([]byte(message), []byte("")))
}

func HeaderColor(message string) string {
	return ColorizeBold(message, defaultFgColor)
}

func CommandColor(message string) string {
	return ColorizeBold(message, yellow)
}

func StoppedColor(message string) string {
	return ColorizeBold(message, grey)
}

func AdvisoryColor(message string) string {
	return ColorizeBold(message, yellow)
}

func CrashedColor(message string) string {
	return ColorizeBold(message, red)
}

func FailureColor(message string) string {
	return ColorizeBold(message, red)
}

func SuccessColor(message string) string {
	return ColorizeBold(message, green)
}

func EntityNameColor(message string) string {
	return ColorizeBold(message, cyan)
}

func PromptColor(message string) string {
	return ColorizeBold(message, cyan)
}

func TableContentHeaderColor(message string) string {
	return ColorizeBold(message, cyan)
}

func WarningColor(message string) string {
	return ColorizeBold(message, magenta)
}

func LogStdoutColor(message string) string {
	return message
}

func LogStderrColor(message string) string {
	return Colorize(message, red)
}

func LogHealthHeaderColor(message string) string {
	return Colorize(message, grey)
}

func LogAppHeaderColor(message string) string {
	return ColorizeBold(message, yellow)
}

func LogSysHeaderColor(message string) string {
	return ColorizeBold(message, cyan)
}

func isTerminal() bool {
	return terminal.IsTerminal(int(os.Stdout.Fd()))
}
