package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmptyMigrationDirectoryDoesNotConnect(t *testing.T) {
	t.Setenv("MIGRATION_DATABASE_URL", "invalid-secret")
	for _, action := range []string{"status", "up", "down"} {
		var out bytes.Buffer
		if err := run(context.Background(), []string{action}, t.TempDir(), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "no database connection") {
			t.Fatal("must explain empty migration state")
		}
	}
}

func TestCreateThenRequireDirectMigrationURL(t *testing.T) {
	dir := t.TempDir()
	if err := run(context.Background(), []string{"create", "create_example"}, dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.sql"))
	if len(files) != 1 {
		t.Fatal("expected one migration template")
	}
	body, err := os.ReadFile(files[0])
	if err != nil || !strings.Contains(string(body), "-- +goose Up") || !strings.Contains(string(body), "-- +goose Down") {
		t.Fatal("invalid Goose template")
	}
	t.Setenv("DATABASE_URL", "postgresql://test:secret@runtime-host/db")
	t.Setenv("MIGRATION_DATABASE_URL", "")
	if err := run(context.Background(), []string{"up"}, dir, io.Discard); err == nil || !strings.Contains(err.Error(), "MIGRATION_DATABASE_URL") {
		t.Fatal("must not fall back to runtime database")
	}
	t.Setenv("MIGRATION_DATABASE_URL", "postgresql://test:secret@ep-example-pooler.neon.tech/db")
	if err := run(context.Background(), []string{"up"}, dir, io.Discard); err == nil || !strings.Contains(err.Error(), "direct Neon URL") {
		t.Fatal("must reject session lock through pooled endpoint")
	}
}

func TestInvalidArgumentsDoNotCreateFiles(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{}, {"reset"}, {"up", "extra"}, {"create", "../escape"}} {
		if err := run(context.Background(), args, dir, io.Discard); err == nil {
			t.Fatal("invalid arguments accepted")
		}
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("invalid commands must not write files")
	}
}
