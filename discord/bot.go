package discord

import (
	"biyobot/configs"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session   *discordgo.Session
	AppConfig *configs.AppConfig
}

func NewDiscordBot(conf *configs.AppConfig) (*DiscordBot) {
	session, err := discordgo.New("Bot " + conf.DiscordToken)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}
	return &DiscordBot{
		Session: session,
		AppConfig: conf,
	}
}
func (b *DiscordBot) Start() {
	// discord bot client
	b.Session.AddHandler(b.onReady)
	b.Session.AddHandler(b.onMessageCreate)
	b.Session.AddHandler(b.onGuildCreate)

	b.Session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent | discordgo.IntentsDirectMessages

	err := b.Session.Open()
	if err != nil {
		log.Println("Error opening connection:", err)
		os.Exit(1)
	}
	defer b.Session.Close()

	fmt.Println("Bot is running. Press Ctrl+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

func (b *DiscordBot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as: %s#%s\n", event.User.Username, event.User.Discriminator)
}

func (b *DiscordBot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch m.Content {
	case "!ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	case "!hello":
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello, %s!", m.Author.Username))
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

func (b *DiscordBot) onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.ID != b.AppConfig.DiscordMasterServerId {
		log.Printf("✗ Bot was added to unauthorized server: %s (ID: %s) - leaving immediately\n",
			event.Guild.Name, event.Guild.ID)
		s.GuildLeave(event.Guild.ID)
	} else {
		log.Printf("✓ Bot added to allowed server: %s\n", event.Guild.Name)
	}
}
