package configs

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DiscordToken string
	DiscordMasterServerId string
	DiscordSrvSchedulerCid string
}

func NewAppConfig() (*AppConfig, error) {
	// env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	} else {
		log.Println(".env loaded successfully")
	}

	var missingVars []string
	discordToken := os.Getenv("DISCORD_BOT_TOKEN")
	if discordToken == "" {
		missingVars = append(missingVars, "DISCORD_BOT_TOKEN")
	}
	discordMasterServerId := os.Getenv("DISCORD_MASTER_SERVER_ID")
	if discordMasterServerId == "" {
		missingVars = append(missingVars, "DISCORD_MASTER_SERVER_ID")
	}
	discordServiceSchedulerCid := os.Getenv("DISCORD_SERVICE_SCHEDULER_CID")
	if discordServiceSchedulerCid  == "" {
		missingVars = append(missingVars, "DISCORD_SERVICE_SCHEDULER_CID")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s",
			strings.Join(missingVars, ", "))
	}
	return &AppConfig{
		DiscordToken: discordToken,
		DiscordMasterServerId: discordMasterServerId,
		DiscordSrvSchedulerCid: discordServiceSchedulerCid,
	}, nil
}
