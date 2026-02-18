package main

import (
	"biyobot/configs"
	"biyobot/discord"
	"biyobot/services"
	"biyobot/services/currency_conversion"
	"biyobot/services/database"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ollama/ollama/api"
)

func main() {
	// appConf, err := configs.NewAppConfig()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// ollama client
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("client: %v", client)

	// // startup db services
	// dbm := database.NewDatabaseManager()
	// // TODO make a better repo collection system
	// notifyRepo := database.NewNotificationsRepo(dbm)

	// register services
	reg := services.NewRegistry()
	reg.Register("currency_converter", &currency_conversion.Service{})
	reg.Register("pythonService", &services.ExternalRunner{
		Executable: "external/test/venv/bin/python3",
		Args:       []string{"external/test/test.py"},
		Timeout:    10 * time.Second,
	})
	// // golang service sample
	// convert_input, _ := json.Marshal(map[string]any{"from": "USD", "to": "JPY", "amount": "15.25"})
	// convert_result := reg.Run("currency_converter", convert_input)
	// if convert_result.OK == false {
	// 	fmt.Printf("Error: %s", convert_result.Error)
	// } else {
	// 	fmt.Printf("Result: %s\n", string(convert_result.Data))
	// }
	// // python service sample
	// py_input, _ := json.Marshal(map[string]any{"name": "Bob"})
	// py_result := reg.Run("pythonService", py_input)
	// if py_result.OK == false {
	// 	fmt.Printf("Error: %s", py_result.Error)
	// } else {
	// 	fmt.Printf("Result: %s\n", string(py_result.Data))
	// }
	//
	// discordBot := discord.NewDiscordBot(appConf, reg)
	// discordBot.Start()
}

