package main

import (
	"biyobot/services"
	"biyobot/services/currency_conversion"
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	reg := services.NewRegistry()
	reg.Register("currencyConverter", &currency_conversion.Service{})
	reg.Register("pythonService", &services.ExternalRunner{
		Executable: "external/test/venv/bin/python3",
		Args:       []string{"external/test/test.py"},
		Timeout:    10 * time.Second,
	})

	// golang service sample
	convert_input, _ := json.Marshal(map[string]any{"from": "USD", "to": "JPY", "amount": "15.25"})
	convert_result := reg.Run("currencyConverter", convert_input)
	if convert_result.OK == false {
		fmt.Printf("Error: %s", convert_result.Error)
	} else {
		fmt.Printf("Result: %s\n", string(convert_result.Data))
	}

	// python service sample
	py_input, _ := json.Marshal(map[string]any{"name": "Bob"})
	py_result := reg.Run("pythonService", py_input)
	if py_result.OK == false {
		fmt.Printf("Error: %s", py_result.Error)
	} else {
		fmt.Printf("Result: %s\n", string(py_result.Data))
	}
}
