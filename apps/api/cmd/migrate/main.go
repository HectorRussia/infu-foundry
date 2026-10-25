package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"infu-foundry/apps/api/internal/database"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], "migrations", os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, dir string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: migrate status|up|down|create <name>")
	}
	action := args[0]
	if action == "create" {
		if len(args) != 2 || !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(args[1]) {
			return errors.New("migration name must use lowercase letters, digits and underscores")
		}
		return goose.Create(nil, dir, args[1], "sql")
	}
	if len(args) != 1 || (action != "status" && action != "up" && action != "down") {
		return errors.New("supported commands: status, up, down, create <name>")
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("migration directory not found: %s", dir)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Fprintln(out, "No SQL migrations yet; no database connection or schema changes made.")
		return nil
	}
	dsn := os.Getenv("MIGRATION_DATABASE_URL")
	if dsn == "" {
		return errors.New("MIGRATION_DATABASE_URL is required; use the direct PostgreSQL URL")
	}
	if err := database.ValidateURL(dsn); err != nil {
		return err
	}
	parsed, _ := url.Parse(dsn)
	if strings.Contains(parsed.Hostname(), "-pooler") {
		return errors.New("migrations require the direct Neon URL, not a -pooler URL")
	}
	db, err := database.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return errors.New("cannot configure migration lock")
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db.SQL, os.DirFS(dir),
		goose.WithSessionLocker(locker), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return errors.New("cannot load migrations; check SQL filenames and Goose annotations")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	switch action {
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return errors.New("cannot read migration status")
		}
		for _, status := range statuses {
			fmt.Fprintf(out, "%d\t%s\t%s\n", status.Source.Version, status.State, status.Source.Path)
		}
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return errors.New("migration failed; inspect the migration and database before retrying")
		}
		fmt.Fprintf(out, "Applied %d migration(s).\n", len(results))
	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return errors.New("rollback failed or no applied migration is available")
		}
		fmt.Fprintf(out, "Rolled back migration %d.\n", result.Source.Version)
	}
	return nil
}
