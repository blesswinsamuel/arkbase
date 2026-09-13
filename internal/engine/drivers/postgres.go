package drivers

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/blesswinsamuel/arkbase/internal/config"
)

type PostgresDriver struct{}

func (p *PostgresDriver) Engine() string {
	return "postgres"
}

func (p *PostgresDriver) Dump(ctx context.Context, dbCfg config.DatabaseConfig, w io.Writer, logWriter io.Writer) error {
	var args []string

	if dbCfg.URI != "" {
		args = append(args, "--dbname", dbCfg.URI)
	} else {
		if dbCfg.Host != "" {
			args = append(args, "-h", dbCfg.Host)
		}
		if dbCfg.Port != 0 {
			args = append(args, "-p", strconv.Itoa(dbCfg.Port))
		}
		if dbCfg.Username != "" {
			args = append(args, "-U", dbCfg.Username)
		}
		if dbCfg.Database != "" {
			args = append(args, "-d", dbCfg.Database)
		}
	}

	// Standard clean flags for logical backups
	args = append(args, "--clean", "--if-exists", "--no-owner", "--no-privileges")
	args = append(args, dbCfg.Options...)

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = os.Environ()
	if dbCfg.Password != "" {
		cmd.Env = append(cmd.Env, "PGPASSWORD="+dbCfg.Password)
	}
	if dbCfg.SSLMode != "" {
		cmd.Env = append(cmd.Env, "PGSSLMODE="+dbCfg.SSLMode)
	}

	cmd.Stderr = logWriter

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pg_dump stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pg_dump start: %w", err)
	}

	// Gzip compress stream
	gw := gzip.NewWriter(w)
	if _, err := io.Copy(gw, stdout); err != nil {
		_ = gw.Close()
		return fmt.Errorf("compress dump stream: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump execution failed: %w", err)
	}

	return nil
}

func (p *PostgresDriver) Restore(ctx context.Context, dbCfg config.DatabaseConfig, r io.Reader, logWriter io.Writer) error {
	var args []string

	if dbCfg.URI != "" {
		args = append(args, "--dbname", dbCfg.URI)
	} else {
		if dbCfg.Host != "" {
			args = append(args, "-h", dbCfg.Host)
		}
		if dbCfg.Port != 0 {
			args = append(args, "-p", strconv.Itoa(dbCfg.Port))
		}
		if dbCfg.Username != "" {
			args = append(args, "-U", dbCfg.Username)
		}
		if dbCfg.Database != "" {
			args = append(args, "-d", dbCfg.Database)
		}
	}

	// Decompress stream
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer gr.Close()

	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = os.Environ()
	if dbCfg.Password != "" {
		cmd.Env = append(cmd.Env, "PGPASSWORD="+dbCfg.Password)
	}
	if dbCfg.SSLMode != "" {
		cmd.Env = append(cmd.Env, "PGSSLMODE="+dbCfg.SSLMode)
	}

	cmd.Stdin = gr
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore execution failed: %w", err)
	}

	return nil
}
