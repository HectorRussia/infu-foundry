// Command createuser adds a login account, or resets its password with -reset.
// The password is read from stdin so it never appears in argv or shell history.
//
//	pnpm db:create-user -email horn@example.com -name Horn
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"infu-foundry/apps/api/internal/auth"
	"infu-foundry/apps/api/internal/database"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "createuser:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("createuser", flag.ContinueOnError)
	email := flags.String("email", "", "login email (required)")
	name := flags.String("name", "", "display name (required for new users)")
	reset := flags.Bool("reset", false, "reset the password of an existing user")
	if err := flags.Parse(args); err != nil {
		return err
	}
	*email = strings.TrimSpace(*email)
	*name = strings.TrimSpace(*name)
	if *email == "" || (!*reset && *name == "") {
		return errors.New("-email and -name are required (-name is optional with -reset)")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}

	fmt.Fprint(stdout, "password: ")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	hash, err := auth.HashPassword(strings.TrimRight(line, "\r\n"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if *reset {
		result := db.ORM.WithContext(ctx).Exec(
			`UPDATE users SET password_hash = ?, updated_at = now() WHERE lower(email) = lower(?)`, hash, *email,
		)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("no user with email %s", *email)
		}
		fmt.Fprintf(stdout, "\npassword reset for %s\n", *email)
		return nil
	}
	err = db.ORM.WithContext(ctx).Exec(
		`INSERT INTO users (email, display_name, password_hash) VALUES (?, ?, ?)`, *email, *name, hash,
	).Error
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return fmt.Errorf("%s already exists; use -reset to change its password", *email)
		}
		return err
	}
	fmt.Fprintf(stdout, "\ncreated %s (%s)\n", *email, *name)
	return nil
}
