package logger

import (
	"os"

	"github.com/morganhein/gondi/schema"
	"github.com/op/go-logging"
)

var Log schema.Logger

func init() {
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfile} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)

	Log = logging.MustGetLogger("mainStdOut")
	backend := logging.NewLogBackend(os.Stderr, "", 0)

	backendFormatter := logging.NewBackendFormatter(backend, format)

	logging.SetBackend(backendFormatter)
}
