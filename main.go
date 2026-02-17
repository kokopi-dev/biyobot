package main

import (
	"biyobot/services"
	"biyobot/services/currency_conversion"
	"encoding/json"
	"fmt"
)

func main() {
	reg := services.NewRegistry()
	reg.Register("currencyConverter", &currency_conversion.Service{})
	convert_input, _ := json.Marshal(map[string]any{"from": "USD", "to": "JPY", "amount": "15.25"})
	convert_result := reg.Run("currencyConverter", convert_input)
	if convert_result.OK == false {
		fmt.Printf("Error: %s", convert_result.Error)
	} else {
		fmt.Printf("Result: %s\n", string(convert_result.Data))
	}
}
