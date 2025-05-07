package db

import (
	"github.com/grassrootseconomics/go-vise/logging"
)

var (
	logg logging.Logger = logging.NewVanilla().WithDomain("db")
)
