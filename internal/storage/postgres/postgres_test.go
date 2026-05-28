package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapCreateUserError(t *testing.T) {
	t.Run("unique violation", func(t *testing.T) {
		err := mapCreateUserError(&pgconn.PgError{Code: "23505"})
		if !errors.Is(err, ErrLoginTaken) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("other error", func(t *testing.T) {
		inner := errors.New("connection refused")
		err := mapCreateUserError(inner)
		if errors.Is(err, ErrLoginTaken) {
			t.Fatal("unexpected ErrLoginTaken")
		}
		if err.Error() != fmt.Sprintf("insert user: %v", inner) {
			t.Fatalf("got %q", err.Error())
		}
	})
}
