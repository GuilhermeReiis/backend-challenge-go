package wager

import (
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

type Snapshot struct {
	ID                             string
	Origin                         Origin
	ProviderID                     string
	ExternalTransactionID          string
	IdempotencyKey                 string
	PayloadHash                    string
	WalletID                       string
	PlayerID                       string
	RoundID                        string
	GameID                         string
	Kind                           Kind
	Money                          money.Money
	ReferenceExternalTransactionID string
	ReferenceTransactionID         string
	Status                         Status
	FailureCode                    string
	ResultBalance                  money.Money
	ProcessedAt                    time.Time
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

func Rehydrate(state Snapshot) (*Transaction, error) {
	err := validateSnapshot(&state)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		id:                             state.ID,
		origin:                         state.Origin,
		providerID:                     state.ProviderID,
		externalTransactionID:          state.ExternalTransactionID,
		idempotencyKey:                 state.IdempotencyKey,
		payloadHash:                    state.PayloadHash,
		walletID:                       state.WalletID,
		playerID:                       state.PlayerID,
		roundID:                        state.RoundID,
		gameID:                         state.GameID,
		kind:                           state.Kind,
		amount:                         state.Money,
		referenceExternalTransactionID: state.ReferenceExternalTransactionID,
		referenceTransactionID:         state.ReferenceTransactionID,
		status:                         state.Status,
		failureCode:                    state.FailureCode,
		resultBalance:                  state.ResultBalance,
		processedAt:                    state.ProcessedAt.UTC(),
		createdAt:                      state.CreatedAt.UTC(),
		updatedAt:                      state.UpdatedAt.UTC(),
	}, nil
}

func validateSnapshot(state *Snapshot) error {
	if state == nil {
		return ErrInvalidTransaction
	}

	err := validateIdentity(state)
	if err != nil {
		return err
	}

	err = validateTimestamps(state)
	if err != nil {
		return err
	}

	err = validateOrigin(state)
	if err != nil {
		return err
	}

	err = validateMoney(state)
	if err != nil {
		return err
	}

	err = validateReferences(state)
	if err != nil {
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

		if !isValidExternalKind(s) {
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

func isValidExternalKind(s *Snapshot) bool {
	switch s.Kind {
	case Bet, Win, Loss, Refund, Rollback:
		return true
	}

	return false
}

func validateMoney(s *Snapshot) error {
	if s.Money.Currency() == "" {
		return money.ErrInvalidMoney
	}

	amount := s.Money.Amount()

	if s.Kind == Loss {
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

	switch s.Kind {
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

	if internal != "" {
		if external == "" || strings.TrimSpace(internal) == "" {
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

	_, err := s.Money.Compare(s.ResultBalance)
	if err != nil {
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
