package drivers

import (
	"context"
	"fmt"
	"io"

	"github.com/blesswinsamuel/arkbase/internal/config"
)

type DatabaseDriver interface {
	Engine() string
	Dump(ctx context.Context, dbCfg config.DatabaseConfig, w io.Writer, logWriter io.Writer) error
	Restore(ctx context.Context, dbCfg config.DatabaseConfig, r io.Reader, logWriter io.Writer) error
}

func GetDriver(engine string) (DatabaseDriver, error) {
	switch engine {
	case "postgres", "postgresql":
		return &PostgresDriver{}, nil
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", engine)
	}
}
