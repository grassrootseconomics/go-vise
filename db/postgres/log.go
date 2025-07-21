package postgres

import (
	slogging "github.com/grassrootseconomics/go-vise/slog"
)

// SetLogger sets the logger for the Postgres backend.
func (pdb *pgDb) SetLogger(logg slogging.Logger) {
	pdb.logg = logg
}
