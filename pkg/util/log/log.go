package log

import (
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

var defaultLog Logger = &stdoutLogger{
	survey: NewSurvey(),
	level: logrus.DebugLevel,
}

func PrintLogo() {
	logo := `
  ____ _            _   _
 / ___(_)_ __   ___| |_| |
| |  _| | '_ \ / __| __| |
| |_| | | | | | (__| |_| |
 \____|_|_| |_|\___|\__|_|`

	stdout.Write([]byte(ansi.Color(logo + "\r\n\r\n", "cyan+b")))
}

func GetInstance() Logger {
	return defaultLog
}