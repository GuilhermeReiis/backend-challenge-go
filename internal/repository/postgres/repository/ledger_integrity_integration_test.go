//go:build integration

package repository_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type ledgerIntegrityRecords struct {
	wallet     models.Wallet
	operation  models.Wager
	references []models.Wager
	entry      models.Ledger
}

func ledgerIntegrityFixture(t *testing.T, kind string, referenceKind string) ledgerIntegrityRecords {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	w := models.Wallet{
		ID: testID(t), PlayerID: testID(t), Currency: "BRL", BalanceMinor: 1000,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	newOperation := func(kind string, reference *models.Wager) models.Wager {
		result := int64(1000)
		operation := models.Wager{
			ID: testID(t), Origin: "EXTERNAL", Kind: kind, Status: "PROCESSED",
			WalletID: w.ID, PlayerID: w.PlayerID, Currency: w.Currency, AmountMinor: 100,
			ResultBalanceMinor: &result, ProcessedAt: &now, CreatedAt: now, UpdatedAt: now,
		}

		if kind == "OPENING" {
			operation.Origin = "INTERNAL"
			operation.AmountMinor = result
		} else {
			provider, externalID, key := "provider-a", testID(t), testID(t)
			hash, round, game := strings.Repeat("a", 64), "round-1", "game-1"
			operation.ProviderID, operation.ExternalTransactionID, operation.IdempotencyKey = &provider, &externalID, &key
			operation.PayloadHash, operation.RoundID, operation.GameID = &hash, &round, &game
		}

		if kind == "LOSS" {
			operation.AmountMinor = 0
		}

		if reference != nil {
			operation.ReferenceTransactionID = &reference.ID
			operation.ReferenceExternalTransactionID = reference.ExternalTransactionID

			if reference.ExternalTransactionID == nil {
				externalID := testID(t)
				operation.ReferenceExternalTransactionID = &externalID
			}
		}

		return operation
	}
	references := []models.Wager{}
	var reference *models.Wager

	if referenceKind != "" {
		var original *models.Wager

		if referenceKind == "REFUND" || referenceKind == "ROLLBACK" {
			bet := newOperation("BET", nil)
			references = append(references, bet)
			original = &bet
		}

		value := newOperation(referenceKind, original)
		references = append(references, value)
		reference = &value
	}

	operation := newOperation(kind, reference)
	direction := "CREDIT"

	if kind == "BET" || (kind == "ROLLBACK" && (referenceKind == "WIN" || referenceKind == "REFUND")) {
		direction = "DEBIT"
	}

	before := *operation.ResultBalanceMinor - operation.AmountMinor

	if direction == "DEBIT" {
		before = *operation.ResultBalanceMinor + operation.AmountMinor
	}

	return ledgerIntegrityRecords{
		wallet: w, operation: operation, references: references,
		entry: models.Ledger{
			ID: testID(t), WalletID: w.ID, TransactionID: operation.ID, TransactionStatus: "PROCESSED",
			Direction: direction, Currency: w.Currency, AmountMinor: operation.AmountMinor,
			BalanceBeforeMinor: before, BalanceAfterMinor: *operation.ResultBalanceMinor, CreatedAt: now,
		},
	}
}

// These inserts bypass repositories and domain validation to exercise the schema.
func insertIntegrityWallet(db *gorm.DB, w models.Wallet) error {
	return db.Exec(`INSERT INTO wallets (id, player_id, currency, balance_minor, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, w.ID, w.PlayerID, w.Currency, w.BalanceMinor, w.Version, w.CreatedAt, w.UpdatedAt).Error
}

func insertIntegrityWager(db *gorm.DB, w models.Wager) error {
	return db.Exec(`INSERT INTO wager_transactions (
		id, origin, kind, status, wallet_id, player_id, currency, amount_minor,
		provider_id, external_transaction_id, idempotency_key, payload_hash, round_id, game_id,
		reference_external_transaction_id, reference_transaction_id, failure_code, result_balance_minor,
		processed_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.Origin, w.Kind, w.Status, w.WalletID, w.PlayerID, w.Currency, w.AmountMinor,
		w.ProviderID, w.ExternalTransactionID, w.IdempotencyKey, w.PayloadHash, w.RoundID, w.GameID,
		w.ReferenceExternalTransactionID, w.ReferenceTransactionID, w.FailureCode, w.ResultBalanceMinor,
		w.ProcessedAt, w.CreatedAt, w.UpdatedAt).Error
}

func insertIntegrityLedger(db *gorm.DB, entry models.Ledger) error {
	return db.Exec(`INSERT INTO wallet_ledger_entries (
		id, wallet_id, transaction_id, transaction_status, direction, currency, amount_minor,
		balance_before_minor, balance_after_minor, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, entry.ID, entry.WalletID, entry.TransactionID,
		entry.TransactionStatus, entry.Direction, entry.Currency, entry.AmountMinor,
		entry.BalanceBeforeMinor, entry.BalanceAfterMinor, entry.CreatedAt).Error
}

func seedIntegrityOperation(db *gorm.DB, records ledgerIntegrityRecords) error {
	if err := insertIntegrityWallet(db, records.wallet); err != nil {
		return err
	}

	for _, reference := range records.references {
		if err := insertIntegrityWager(db, reference); err != nil {
			return err
		}
	}

	return insertIntegrityWager(db, records.operation)
}

func requireIntegrityError(t *testing.T, err error, code string, constraint string) {
	t.Helper()
	var pgError *pgconn.PgError

	if !errors.As(err, &pgError) || pgError.Code != code {
		t.Fatalf("expected PostgreSQL error %s, got %v", code, err)
	}

	if constraint != "" && pgError.ConstraintName != constraint {
		t.Fatalf("expected constraint %s, got %s", constraint, pgError.ConstraintName)
	}
}

func TestLedgerOperationIntegrity(t *testing.T) {
	db, _ := repositoryTestDatabase(t)
	directions := []struct {
		name      string
		kind      string
		reference string
	}{
		{"opening credit", "OPENING", ""},
		{"bet debit", "BET", ""},
		{"win credit", "WIN", ""},
		{"refund credit", "REFUND", "BET"},
		{"rollback of bet credit", "ROLLBACK", "BET"},
		{"rollback of win debit", "ROLLBACK", "WIN"},
		{"rollback of refund debit", "ROLLBACK", "REFUND"},
	}

	for _, tc := range directions {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("accepts compatible direction and result", func(t *testing.T) {
				records := ledgerIntegrityFixture(t, tc.kind, tc.reference)

				if err := db.Transaction(func(tx *gorm.DB) error {
					if err := seedIntegrityOperation(tx, records); err != nil {
						return err
					}

					return insertIntegrityLedger(tx, records.entry)
				}); err != nil {
					t.Fatal(err)
				}

				var stored models.Ledger

				if err := db.Where("id = ?", records.entry.ID).Take(&stored).Error; err != nil {
					t.Fatal(err)
				}

				if stored.Direction != records.entry.Direction || stored.BalanceAfterMinor != records.entry.BalanceAfterMinor || stored.AmountMinor != records.entry.AmountMinor {
					t.Error("expected exact ledger direction, result and amount")
				}
			})

			t.Run("rejects opposite direction even with valid arithmetic", func(t *testing.T) {
				records := ledgerIntegrityFixture(t, tc.kind, tc.reference)
				records.entry.Direction = "DEBIT"
				records.entry.BalanceBeforeMinor = records.entry.BalanceAfterMinor + records.entry.AmountMinor

				if tc.kind == "BET" || (tc.kind == "ROLLBACK" && tc.reference != "BET") {
					records.entry.Direction = "CREDIT"
					records.entry.BalanceBeforeMinor = records.entry.BalanceAfterMinor - records.entry.AmountMinor
				}

				err := db.Transaction(func(tx *gorm.DB) error {
					if err := seedIntegrityOperation(tx, records); err != nil {
						return err
					}

					return insertIntegrityLedger(tx, records.entry)
				})
				requireIntegrityError(t, err, "23514", "ledger_operation_direction_check")

				if countRows(t, db, "wallets", "id", records.wallet.ID) != 0 || countRows(t, db, "wager_transactions", "id", records.operation.ID) != 0 || countRows(t, db, "wallet_ledger_entries", "id", records.entry.ID) != 0 {
					t.Error("expected invalid ledger to roll back all writes")
				}
			})
		})
	}

	t.Run("rejects a result mismatch even with valid arithmetic", func(t *testing.T) {
		records := ledgerIntegrityFixture(t, "WIN", "")
		records.entry.BalanceBeforeMinor++
		records.entry.BalanceAfterMinor++

		if err := seedIntegrityOperation(db, records); err != nil {
			t.Fatal(err)
		}

		requireIntegrityError(t, insertIntegrityLedger(db, records.entry), "23514", "ledger_operation_result_check")

		if countRows(t, db, "wallet_ledger_entries", "id", records.entry.ID) != 0 {
			t.Error("expected rejected ledger not to be persisted")
		}
	})

	for _, kind := range []string{"LOSS", "OPENING", "ROLLBACK"} {
		t.Run("rejects rollback direction for reference "+kind, func(t *testing.T) {
			records := ledgerIntegrityFixture(t, "ROLLBACK", kind)

			if err := seedIntegrityOperation(db, records); err != nil {
				t.Fatal(err)
			}

			requireIntegrityError(t, insertIntegrityLedger(db, records.entry), "23514", "ledger_operation_direction_check")
		})
	}

	t.Run("preserves existing ledger constraints", func(t *testing.T) {
		cases := []struct {
			name       string
			change     func(*ledgerIntegrityRecords)
			code       string
			constraint string
		}{
			{"amount", func(r *ledgerIntegrityRecords) { r.entry.AmountMinor++; r.entry.BalanceBeforeMinor-- }, "23503", "ledger_transaction_fk"},
			{"currency", func(r *ledgerIntegrityRecords) { r.entry.Currency = "USD" }, "23503", ""},
			{"missing operation", func(r *ledgerIntegrityRecords) { r.entry.TransactionID = testID(t) }, "23503", "ledger_transaction_fk"},
			{"status", func(r *ledgerIntegrityRecords) { r.entry.TransactionStatus = "REJECTED" }, "23514", "ledger_transaction_status_check"},
			{"arithmetic", func(r *ledgerIntegrityRecords) { r.entry.BalanceBeforeMinor++ }, "23514", "ledger_balance_math_check"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				records := ledgerIntegrityFixture(t, "WIN", "")
				tc.change(&records)

				if err := seedIntegrityOperation(db, records); err != nil {
					t.Fatal(err)
				}

				requireIntegrityError(t, insertIntegrityLedger(db, records.entry), tc.code, tc.constraint)
			})
		}

		t.Run("wallet binding", func(t *testing.T) {
			records := ledgerIntegrityFixture(t, "WIN", "")
			other := ledgerIntegrityFixture(t, "WIN", "")

			if err := seedIntegrityOperation(db, records); err != nil {
				t.Fatal(err)
			}

			if err := insertIntegrityWallet(db, other.wallet); err != nil {
				t.Fatal(err)
			}

			records.entry.WalletID = other.wallet.ID
			requireIntegrityError(t, insertIntegrityLedger(db, records.entry), "23503", "ledger_transaction_fk")
		})

		t.Run("uniqueness", func(t *testing.T) {
			records := ledgerIntegrityFixture(t, "WIN", "")

			if err := seedIntegrityOperation(db, records); err != nil {
				t.Fatal(err)
			}

			if err := insertIntegrityLedger(db, records.entry); err != nil {
				t.Fatal(err)
			}

			records.entry.ID = testID(t)
			requireIntegrityError(t, insertIntegrityLedger(db, records.entry), "23505", "ledger_wallet_transaction_unique")
		})
	})
}

func TestWagerWalletOwnershipIntegrity(t *testing.T) {
	db, _ := repositoryTestDatabase(t)

	t.Run("rejects insertion for another player", func(t *testing.T) {
		records := ledgerIntegrityFixture(t, "WIN", "")

		if err := insertIntegrityWallet(db, records.wallet); err != nil {
			t.Fatal(err)
		}

		records.operation.PlayerID = testID(t)
		requireIntegrityError(t, insertIntegrityWager(db, records.operation), "23503", "wager_wallet_player_currency_fk")
	})

	for _, target := range []string{"operation player", "operation wallet", "wallet player"} {
		t.Run("rejects update of "+target, func(t *testing.T) {
			records := ledgerIntegrityFixture(t, "WIN", "")
			records.operation.Status = "PENDING"
			records.operation.ResultBalanceMinor, records.operation.ProcessedAt = nil, nil

			if err := seedIntegrityOperation(db, records); err != nil {
				t.Fatal(err)
			}

			var err error
			switch target {
			case "operation player":
				err = db.Exec("UPDATE wager_transactions SET player_id = ? WHERE id = ?", testID(t), records.operation.ID).Error
			case "operation wallet":
				other := ledgerIntegrityFixture(t, "WIN", "")

				if err := insertIntegrityWallet(db, other.wallet); err != nil {
					t.Fatal(err)
				}

				err = db.Exec("UPDATE wager_transactions SET wallet_id = ? WHERE id = ?", other.wallet.ID, records.operation.ID).Error
			case "wallet player":
				err = db.Exec("UPDATE wallets SET player_id = ? WHERE id = ?", testID(t), records.wallet.ID).Error
			}
			requireIntegrityError(t, err, "23503", "wager_wallet_player_currency_fk")

			if err := db.Exec("UPDATE wager_transactions SET player_id = ? WHERE id = ?", records.wallet.PlayerID, records.operation.ID).Error; err != nil {
				t.Fatal("expected an update preserving wallet ownership to succeed:", err)
			}
		})
	}
}

func applyLedgerIntegrityMigration(t *testing.T, db *gorm.DB, direction string) error {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "migrations", "000006_enforce_ledger_operation_integrity."+direction+".sql")
	migration, err := os.ReadFile(path)

	if err != nil {
		t.Fatal(err)
	}

	pool, err := db.DB()

	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := pool.Conn(ctx)

	if err != nil {
		t.Fatal(err)
	}

	defer connection.Close()
	_, err = connection.ExecContext(ctx, string(migration))

	if err != nil {
		if _, rollbackErr := connection.ExecContext(context.Background(), "ROLLBACK"); rollbackErr != nil {
			t.Fatal("could not roll back failed migration:", rollbackErr)
		}
	}

	return err
}

func requireOriginalLedgerProtection(t *testing.T, admin *gorm.DB, entryID string) {
	t.Helper()
	for _, statement := range []string{
		"UPDATE wallet_ledger_entries SET balance_after_minor = balance_after_minor WHERE id = ?",
		"DELETE FROM wallet_ledger_entries WHERE id = ?",
	} {
		requireIntegrityError(t, admin.Exec(statement, entryID).Error, "23001", "")
	}

	requireIntegrityError(t, admin.Exec("TRUNCATE wallet_ledger_entries").Error, "23001", "")
}

func TestLedgerIntegrityMigration(t *testing.T) {
	t.Run("reapplies over valid data without changing history or original protection", func(t *testing.T) {
		db, admin := repositoryTestDatabase(t)

		if err := applyLedgerIntegrityMigration(t, admin, "down"); err != nil {
			t.Fatal(err)
		}

		records := ledgerIntegrityFixture(t, "WIN", "")

		if err := seedIntegrityOperation(db, records); err != nil {
			t.Fatal(err)
		}

		if err := insertIntegrityLedger(db, records.entry); err != nil {
			t.Fatal(err)
		}

		var before models.Ledger

		if err := db.Where("id = ?", records.entry.ID).Take(&before).Error; err != nil {
			t.Fatal(err)
		}

		requireOriginalLedgerProtection(t, admin, records.entry.ID)

		if err := applyLedgerIntegrityMigration(t, admin, "up"); err != nil {
			t.Fatal(err)
		}

		var after models.Ledger

		if err := db.Where("id = ?", records.entry.ID).Take(&after).Error; err != nil {
			t.Fatal(err)
		}

		if before != after {
			t.Error("expected migration to preserve exact ledger history")
		}

		requireOriginalLedgerProtection(t, admin, records.entry.ID)
		invalid := ledgerIntegrityFixture(t, "BET", "")

		if err := seedIntegrityOperation(db, invalid); err != nil {
			t.Fatal(err)
		}

		invalid.entry.Direction = "CREDIT"
		invalid.entry.BalanceBeforeMinor = invalid.entry.BalanceAfterMinor - invalid.entry.AmountMinor
		requireIntegrityError(t, insertIntegrityLedger(db, invalid.entry), "23514", "ledger_operation_direction_check")
	})

	cases := []struct {
		name       string
		kind       string
		reference  string
		change     func(*ledgerIntegrityRecords)
		withLedger bool
		code       string
		constraint string
	}{
		{"wrong wallet player", "WIN", "", func(r *ledgerIntegrityRecords) { r.operation.PlayerID = testID(t) }, false, "23503", "wager_wallet_player_currency_fk"},
		{"opposite ledger direction", "BET", "", func(r *ledgerIntegrityRecords) {
			r.entry.Direction = "CREDIT"
			r.entry.BalanceBeforeMinor = r.entry.BalanceAfterMinor - r.entry.AmountMinor
		}, true, "23514", "ledger_operation_direction_check"},
		{"incompatible rollback reference", "ROLLBACK", "LOSS", func(r *ledgerIntegrityRecords) {}, true, "23514", "ledger_operation_direction_check"},
		{"different operation result", "WIN", "", func(r *ledgerIntegrityRecords) {
			r.entry.BalanceBeforeMinor++
			r.entry.BalanceAfterMinor++
		}, true, "23514", "ledger_operation_result_check"},
	}

	for _, tc := range cases {
		t.Run("refuses existing data with "+tc.name, func(t *testing.T) {
			db, admin := repositoryTestDatabase(t)

			if err := applyLedgerIntegrityMigration(t, admin, "down"); err != nil {
				t.Fatal(err)
			}

			records := ledgerIntegrityFixture(t, tc.kind, tc.reference)
			tc.change(&records)

			if err := seedIntegrityOperation(db, records); err != nil {
				t.Fatal(err)
			}

			if tc.withLedger {
				if err := insertIntegrityLedger(db, records.entry); err != nil {
					t.Fatal(err)
				}
			}

			requireIntegrityError(t, applyLedgerIntegrityMigration(t, admin, "up"), tc.code, tc.constraint)
			var partiallyApplied bool

			if err := admin.Raw(`SELECT
				to_regprocedure('public.ledger_expected_direction(text,text)') IS NOT NULL
				OR to_regprocedure('public.validate_ledger_operation()') IS NOT NULL
				OR EXISTS (SELECT 1 FROM pg_constraint WHERE conname IN (
					'wallets_id_player_currency_unique', 'wager_wallet_player_currency_fk'
				)) AS partially_applied`).Scan(&partiallyApplied).Error; err != nil {
				t.Fatal(err)
			}

			if partiallyApplied {
				t.Error("expected failed migration to leave no partial schema changes")
			}

			var stored models.Wager

			if err := db.Where("id = ?", records.operation.ID).Take(&stored).Error; err != nil {
				t.Fatal(err)
			}

			if stored.PlayerID != records.operation.PlayerID || *stored.ResultBalanceMinor != *records.operation.ResultBalanceMinor {
				t.Error("expected migration not to correct persisted operation data")
			}

			if tc.withLedger {
				var entry models.Ledger

				if err := db.Where("id = ?", records.entry.ID).Take(&entry).Error; err != nil {
					t.Fatal(err)
				}

				if entry.Direction != records.entry.Direction || entry.BalanceAfterMinor != records.entry.BalanceAfterMinor {
					t.Error("expected migration not to correct financial history")
				}

				requireOriginalLedgerProtection(t, admin, records.entry.ID)
			}
		})
	}
}
