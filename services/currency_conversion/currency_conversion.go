package currency_conversion

import (
	"biyobot/models"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

type Input struct {
	From       string `json:"from"`
	To         string `json:"to"`
	FromAmount string `json:"amount"`
}

type Output struct {
	ConvertedRaw    int64  `json:"converted_raw"`
	ConvertedAmount string `json:"converted_amount"`
}

var currencyDecimals = map[string]int32{
	"JPY": 0,
	"USD": 2,
	"EUR": 2,
}
var currencyUnits = map[string]int64{
	"USD": 100, // 1 USD = 100 cents
	"EUR": 100,
	"JPY": 1, // no minor units
}

func toRaw(amount string, currency string) (int64, error) {
	units, ok := currencyUnits[currency]
	if !ok {
		return 0, fmt.Errorf("unsupported currency: %s", currency)
	}
	parsed, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", amount)
	}
	return int64(parsed * float64(units)), nil
}

func convert(raw int64, fromCurrency string, rate float64, toCurrency string) int64 {
	fromUnits, ok := currencyUnits[fromCurrency]
	if !ok {
		fromUnits = 100
	}
	toUnits, ok := currencyUnits[toCurrency]
	if !ok {
		toUnits = 100
	}
	return int64((float64(raw) / float64(fromUnits)) * rate * float64(toUnits))
}

func toAmount(raw int64, currency string) string {
	decimals, ok := currencyDecimals[currency]
	if !ok {
		decimals = 2
	}
	divisor := math.Pow(10, float64(decimals))
	return fmt.Sprintf("%s %.*f", currency, decimals, float64(raw)/divisor)
}

type Service struct{}

func (s *Service) Run(msg json.RawMessage) models.ServiceResult {
	var input Input
	if err := json.Unmarshal(msg, &input); err != nil {
		return models.Failure("invalid input: " + err.Error())
	}
	if input.FromAmount == "" {
		return models.Failure("`amount` is required")
	}
	if input.From == "" {
		return models.Failure("`from` is required")
	}
	if input.To == "" {
		return models.Failure("`to` is required")
	}
	fromRaw, err := toRaw(input.FromAmount, input.From)
	if err != nil {
		return models.Failure(err.Error())
	}
	convertedRaw := convert(fromRaw, input.From, 149.50, input.To)
	return models.Success(Output{
		ConvertedRaw:    convertedRaw,
		ConvertedAmount: toAmount(convertedRaw, input.To),
	})
}
