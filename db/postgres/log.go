package postgres

import (
	"github.com/grassrootseconomics/go-vise/logging"
	slogging "github.com/grassrootseconomics/go-vise/slog"
)

// SetLogger sets the logger for the Postgres backend.
// If logg is nil, a new instance of Slog is created with trace level options.
func (pdb *pgDb) SetLogger(logg logging.Logger) {
	if logg != nil {
		pdb.logg = logg
		return
	}

	if pdb.logg == nil {
		pdb.logg = slogging.NewSlog(slogging.SlogOpts{
			Component:     "vise.postgresdb",
			LogLevel:      slogging.LevelTrace,
			IncludeSource: true,
		})
	}
}
