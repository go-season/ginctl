package log

import (
	"io"
	"os"

	dockerterm "github.com/docker/docker/pkg/term"
	"k8s.io/kubectl/pkg/util/term"
)

func SetupTTY(stdin io.Reader, stdout io.Writer) term.TTY {
	t := term.TTY{
		Out: stdout,
		In:  stdin,
	}

	if !t.IsTerminalIn() {
		return t
	}

	t.Raw = true

	stdin, stdout, _ = dockerterm.StdStreams()

	if stdin == os.Stdin {
		t.In = stdin
	}

	if stdout == os.Stdout {
		t.Out = stdout
	}

	return t
}
