package currency

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ExchangeRate struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

func GetRate(from, to string) (float64, error) {
	url := fmt.Sprintf("http://frankfurter:8080/v1/latest?from=%s&to=%s", from, to)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var exchangeRate ExchangeRate
	if err := json.NewDecoder(resp.Body).Decode(&exchangeRate); err != nil {
		return 0, err
	}

	rate, ok := exchangeRate.Rates[to]
	if !ok {
		return 0, fmt.Errorf("rate not found for %s", to)
	}

	return rate, nil
}
