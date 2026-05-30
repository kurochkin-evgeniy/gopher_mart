package postgres

import "testing"

func TestNormalizePostgresURI(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "postgresql://user:pass@host/db",
			want: "postgres://user:pass@host/db",
		},
		{
			in:   "postgres://user:pass@host/db",
			want: "postgres://user:pass@host/db",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizePostgresURI(tc.in); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
