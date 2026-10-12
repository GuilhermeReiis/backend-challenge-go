package money

import (
	"encoding/json"
	"fmt"
)

func (m Money) String() string {
	if m.currency == "" {
		return "<invalid money>"
	}

	sign := ""
	magnitude := uint64(m.amount)

	if m.amount < 0 {
		sign = "-"
		magnitude = uint64(-(m.amount + 1)) + 1
	}

	return fmt.Sprintf(
		"%s%d.%02d",
		sign,
		magnitude/100,
		magnitude%100,
	)
}

func (m Money) MarshalJSON() ([]byte, error) {
	if m.currency == "" {
		return nil, ErrInvalidMoney
	}

	return json.Marshal(struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}{
		Amount:   m.String(),
		Currency: m.currency,
	})
}
