package wager_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

func TestPayloadHash(t *testing.T) {
	cases := []struct {
		name    string
		hash    string
		wantErr error
	}{
		{name: "lowercase hexadecimal", hash: strings.Repeat("0123456789abcdef", 4)},
		{name: "all zeros", hash: strings.Repeat("0", 64)},
		{name: "all letters", hash: strings.Repeat("f", 64)},
		{name: "empty", wantErr: wager.ErrInvalidIdentity},
		{name: "blank", hash: " \t", wantErr: wager.ErrInvalidIdentity},
		{name: "too short", hash: strings.Repeat("a", 63), wantErr: wager.ErrInvalidPayloadHash},
		{name: "too long", hash: strings.Repeat("a", 65), wantErr: wager.ErrInvalidPayloadHash},
		{name: "uppercase", hash: strings.Repeat("A", 64), wantErr: wager.ErrInvalidPayloadHash},
		{name: "mixed case", hash: "A" + strings.Repeat("a", 63), wantErr: wager.ErrInvalidPayloadHash},
		{name: "non hexadecimal", hash: strings.Repeat("g", 64), wantErr: wager.ErrInvalidPayloadHash},
		{name: "leading whitespace", hash: " " + strings.Repeat("a", 64), wantErr: wager.ErrInvalidPayloadHash},
		{name: "trailing whitespace", hash: strings.Repeat("a", 64) + "\n", wantErr: wager.ErrInvalidPayloadHash},
		{name: "unicode", hash: strings.Repeat("a", 62) + "é", wantErr: wager.ErrInvalidPayloadHash},
	}
	constructors := []struct {
		name string
		new  func(*testing.T, string) (*wager.Transaction, error)
	}{
		{
			name: "external creation",
			new: func(t *testing.T, hash string) (*wager.Transaction, error) {
				input := validInput(t)
				input.PayloadHash = hash
				return wager.NewExternal(input, validSnapshot(t).CreatedAt)
			},
		},
		{
			name: "rehydration",
			new: func(t *testing.T, hash string) (*wager.Transaction, error) {
				state := validSnapshot(t)
				state.PayloadHash = hash
				return wager.Rehydrate(state)
			},
		},
	}

	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					value, err := constructor.new(t, tc.hash)

					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
					}

					if tc.wantErr != nil {
						if value != nil {
							t.Error("expected no transaction on validation failure")
						}

						return
					}

					if value.PayloadHash() != tc.hash {
						t.Error("expected payload hash to be preserved exactly")
					}
				})
			}
		})
	}
}
