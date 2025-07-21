package fs

import (
	slogging "github.com/grassrootseconomics/go-vise/slog"
)

var (
	logg = slogging.Global.With("component", "fsdb")
)
