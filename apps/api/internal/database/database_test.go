package database

import (
	"context"
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	for _, value := range []string{"", "https://example.com", "postgresql:///db", "postgresql://user:secret%ZZ@localhost/db"} {
		if err := ValidateURL(value); err == nil {
			t.Errorf("invalid URL accepted")
		} else if strings.Contains(err.Error(), "secret") {
			t.Error("error exposed credentials")
		}
	}
	if err := ValidateURL("postgresql://user:password@localhost/db?sslmode=require"); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCancellationDoesNotExposeConnectionDetails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db, err := Open(ctx, "postgresql://test:secret-password@127.0.0.1:1/test?sslmode=disable")
	if err == nil || db != nil {
		t.Fatal("cancelled connection should fail")
	}
	if strings.Contains(err.Error(), "secret-password") || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatal("connection details leaked")
	}
}
