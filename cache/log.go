package cache

import (
	slogging "github.com/grassrootseconomics/go-vise/slog"
)

var logg = slogging.Get().With("component", "cache")
