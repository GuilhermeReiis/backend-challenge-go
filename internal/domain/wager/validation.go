package wager

import (
	"strings"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func validateSnapshot(state *Snapshot) error {
	if state == nil {
		return ErrInvalidTransaction
	}

	if err := validateIdentity(state); err != nil {
		return err
	}

	if err := validateTimestamps(state); err != nil {
		return err
	}

	if err := validateOrigin(state); err != nil {
		return err
	}

	if err := validateAmount(state.Kind, state.Money); err != nil {
		return err
	}

	if err := validateReferences(state); err != nil {
		return err
	}

	return validateStatus(state)
}

func validateIdentity(s *Snapshot) error {
	if hasBlank(s.ID, s.WalletID, s.PlayerID) {
		return ErrInvalidIdentity
	}

	return nil
}

func validateTimestamps(s *Snapshot) error {
	if s.CreatedAt.IsZero() ||
		s.UpdatedAt.IsZero() ||
		s.UpdatedAt.Before(s.CreatedAt) {
		return ErrInvalidTime
	}

	return nil
}

func validateOrigin(s *Snapshot) error {
	metadata := []string{
		s.ProviderID,
		s.ExternalTransactionID,
		s.IdempotencyKey,
		s.PayloadHash,
		s.RoundID,
		s.GameID,
	}

	switch s.Origin {
	case ExternalOrigin:
		if hasBlank(metadata...) {
			return ErrInvalidIdentity
		}

		if !isValidExternalKind(s.Kind) {
			return ErrInvalidKind
		}

	case InternalOrigin:
		if s.Kind != Opening {
			return ErrInvalidKind
		}

		if hasNonEmpty(metadata...) {
			return ErrInvalidTransaction
		}

	default:
		return ErrInvalidTransaction
	}

	return nil
}

func isValidExternalKind(kind Kind) bool {
	switch kind {
	case Bet, Win, Loss, Refund, Rollback:
		return true
	}

	return false
}

func validateAmount(kind Kind, value money.Money) error {
	if value.Currency() == "" {
		return money.ErrInvalidMoney
	}

	amount := value.Amount()

	if kind == Loss {
		if amount != 0 {
			return ErrInvalidAmount
		}
	} else if amount <= 0 {
		return ErrInvalidAmount
	}

	return nil
}

func validateReferences(s *Snapshot) error {
	external := s.ReferenceExternalTransactionID
	internal := s.ReferenceTransactionID

	if err := validateExternalReference(s.Kind, external); err != nil {
		return err
	}

	if internal != "" {
		if external == "" || strings.TrimSpace(internal) == "" {
			return ErrInvalidReference
		}
	}

	return nil
}

func validateExternalReference(kind Kind, external string) error {
	switch kind {
	case Refund, Rollback:
		if strings.TrimSpace(external) == "" {
			return ErrInvalidReference
		}

	case Win:
		if external != "" && strings.TrimSpace(external) == "" {
			return ErrInvalidReference
		}

	default:
		if external != "" {
			return ErrInvalidReference
		}
	}

	return nil
}

func validateStatus(s *Snapshot) error {
	switch s.Status {
	case Processed:
		return validateProcessed(s)

	case Rejected, Failed:
		if strings.TrimSpace(s.FailureCode) == "" {
			return ErrInvalidFailureCode
		}

	case Pending, PendingReference:
		if s.FailureCode != "" {
			return ErrInvalidFailureCode
		}

		if s.ReferenceTransactionID != "" {
			return ErrInvalidReference
		}

		if s.Status == PendingReference &&
			s.ReferenceExternalTransactionID == "" {
			return ErrInvalidReference
		}

	default:
		return ErrInvalidTransaction
	}

	if !s.ProcessedAt.IsZero() ||
		s.ResultBalance.Currency() != "" {
		return ErrInvalidTransaction
	}

	return nil
}

func validateProcessed(s *Snapshot) error {
	if s.FailureCode != "" {
		return ErrInvalidFailureCode
	}

	if s.ProcessedAt.IsZero() ||
		!s.ProcessedAt.Equal(s.UpdatedAt) {
		return ErrInvalidTime
	}

	if _, err := s.Money.Compare(s.ResultBalance); err != nil {
		return err
	}

	if s.ResultBalance.Amount() < 0 {
		return ErrInvalidAmount
	}

	if s.ReferenceExternalTransactionID != "" &&
		strings.TrimSpace(s.ReferenceTransactionID) == "" {
		return ErrInvalidReference
	}

	return nil
}

func hasBlank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}

	return false
}

func hasNonEmpty(values ...string) bool {
	for _, value := range values {
		if value != "" {
			return true
		}
	}

	return false
}
