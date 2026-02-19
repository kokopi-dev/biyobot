package discord

import (
	"biyobot/configs"
	"biyobot/llm"
	"biyobot/models"
	"biyobot/services"
	"biyobot/services/database"
	"biyobot/utils"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type DiscordBot struct {
	Session            *discordgo.Session
	AppConfig          *configs.AppConfig
	Services           *services.Registry
	IntentService      *llm.IntentService
	DiscordMessageRepo *database.DiscordMessageRepo
	NotificationsRepo  *database.NotificationsRepo
}

func NewDiscordBot(conf *configs.AppConfig, services *services.Registry, intentService *llm.IntentService, messageRepo *database.DiscordMessageRepo, notifyRepo *database.NotificationsRepo) *DiscordBot {
	session, err := discordgo.New("Bot " + conf.DiscordToken)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}
	return &DiscordBot{
		Session:            session,
		AppConfig:          conf,
		Services:           services,
		IntentService:      intentService,
		DiscordMessageRepo: messageRepo,
		NotificationsRepo:  notifyRepo,
	}
}
func (b *DiscordBot) Start(ctx context.Context) {
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

	// start background tasks
	fmt.Println("Starting bot background tasks")
	services.StartBackgroundTask(ctx, 180, b.DeleteExpiredMessages)
	services.StartBackgroundTask(ctx, 60, b.HandleNotificationDm)

	fmt.Println("Bot is running. Press Ctrl+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

func (b *DiscordBot) DeleteExpiredMessages(ctx context.Context) {
	expiredMessages, err := b.DiscordMessageRepo.GetAllExpiredMessages()
	if err != nil {
		log.Println("Getting expired messages failed")
		return
	}
	var ids []uuid.UUID
	for _, e := range expiredMessages {
		err := b.Session.ChannelMessageDelete(e.ChannelId, e.MessageId)
		if err == nil {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) != 0 {
		err := b.DiscordMessageRepo.DeleteMessageBatch(ids)
		if err != nil {
			log.Println("Getting expired messages failed")
			return
		}
	}
}

func (b *DiscordBot) HandleNotificationDm(ctx context.Context) {
	expiredNotifications, err := b.NotificationsRepo.GetAllExpiredNotifications()
	if err != nil {
		log.Println("Discord bot failed to get expired notifications")
		return
	}
	if len(expiredNotifications) == 0 {
		return
	}
	var ids []uuid.UUID
	for _, n := range expiredNotifications {
		metadata, err := utils.JsonToStruct[configs.DiscordMetadata](n.Metadata)
		if err != nil {
			log.Printf("failed to parse metadata for notification %s: %v", n.ID, err)
			continue
		}

		channel, err := b.Session.UserChannelCreate(metadata.UserId)
		if err != nil {
			log.Printf("failed to create DM channel for user %s: %v", metadata.UserId, err)
			continue
		}

		content := fmt.Sprintf(
			"🔔 **%s**\n\n%s\n\n⏰ Scheduled for: %s",
			n.Title,
			n.Message,
			n.NotifyAt.Format("Jan 02, 2006 15:04 MST"),
		)

		sentMsg, err := b.Session.ChannelMessageSend(channel.ID, content)
		if err != nil {
			log.Printf("failed to send DM to user %s: %v", metadata.UserId, err)
			continue
		}
		b.tagMessageToBeDeleted(sentMsg, 86400)

		ids = append(ids, n.ID)
	}

	if len(ids) > 0 {
		if err := b.NotificationsRepo.DeleteNotificationBatch(ids); err != nil {
			log.Printf("failed to delete processed notifications: %v", err)
		}
	}
}

func (b *DiscordBot) tagMessageToBeDeleted(msg *discordgo.Message, secondsTillDelete int) error {
	_, err := b.DiscordMessageRepo.AddMessage(database.AddDiscordMessageDto{
		Action:          "delete",
		ChannelId:       msg.ChannelID,
		UserId:          msg.Author.ID,
		MessageId:       msg.ID,
		Content:         msg.Content,
		ExecuteActionOn: utils.JapanTimeNow().Add(time.Duration(secondsTillDelete) * time.Second),
	})
	return err
}

// handles notifications service
func (b *DiscordBot) handleNotifications(intent *llm.IntentResult, discordMeta *configs.DiscordMetadata) {
	metadata, err := utils.StructToJson(discordMeta)
	if err != nil {
		log.Println("failed to serialize discord metadata:", err)
		return
	}

	var replyContent string
	switch intent.Action {
	case "add":
		notifyAt, err := time.Parse(time.RFC3339, utils.ParamString(intent.Params, "notify_at"))
		if err != nil {
			log.Println("failed to parse notify_at:", err)
			return
		}
		title := utils.ParamString(intent.Params, "title")
		_, err = b.NotificationsRepo.AddNotification(database.AddNotificationDto{
			Service:  "scheduler",
			Metadata: metadata,
			NotifyAt: notifyAt,
			Title:    title,
			Message:  utils.ParamString(intent.Params, "description"),
		})
		if err != nil {
			log.Println("failed to add notification:", err)
			return
		}
		replyContent = fmt.Sprintf("✅ Scheduled **%s** for %s", title, notifyAt.Format("Jan 02, 2006 15:04 MST"))
	case "edit":
		notifyAt, err := time.Parse(time.RFC3339, utils.ParamString(intent.Params, "notify_at"))
		if err != nil {
			log.Println("failed to parse notify_at:", err)
			return
		}
		title := utils.ParamString(intent.Params, "title")
		_, err = b.NotificationsRepo.EditNotification(database.EditNotificationDto{
			ID:       utils.ParamString(intent.Params, "notification_id"),
			Service:  "scheduler",
			Metadata: metadata,
			NotifyAt: notifyAt,
			Title:    title,
			Message:  utils.ParamString(intent.Params, "description"),
		})
		if err != nil {
			log.Println("failed to edit notification:", err)
			return
		}
		replyContent = fmt.Sprintf("✏️ Updated **%s** to %s", title, notifyAt.Format("Jan 02, 2006 15:04 MST"))
	case "delete":
		notificationId, err := uuid.Parse(utils.ParamString(intent.Params, "notification_id"))
		if err != nil {
			log.Println("failed to parse notification_id:", err)
			return
		}
		err = b.NotificationsRepo.DeleteNotification(notificationId)
		if err != nil {
			log.Println("failed to delete notification:", err)
			return
		}
		replyContent = fmt.Sprintf("🗑️ Deleted notification `%s`", notificationId)
	}

	b.updateNotifications()
}
func formatNotifications(notifications []models.Notification) string {
	if len(notifications) == 0 {
		return "📭 No upcoming notifications."
	}

	var b strings.Builder
	b.WriteString("📅 **Upcoming Notifications**\n\n")

	for _, n := range notifications {
		fmt.Fprintf(&b, "**%s**\n", n.Title)
		fmt.Fprintf(&b, "📝 %s\n", n.Message)
		fmt.Fprintf(&b, "⏰ %s\n", n.NotifyAt.Format("Jan 02, 2006 15:04 MST"))
		if n.Service != "" {
			fmt.Fprintf(&b, "🔧 %s\n", n.Service)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (b *DiscordBot) getFirstMessageInChannel(channelID string) (*discordgo.Message, error) {
	messages, err := b.Session.ChannelMessages(channelID, 1, "", "", "0")
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return messages[0], nil
}

func (b *DiscordBot) updateNotifications() {
	allNotifications, err := b.NotificationsRepo.GetAllNotifications()
	if err != nil {
		log.Println("Discord bot getting notifications failed.")
		return
	}
	content := formatNotifications(allNotifications)
	firstMsg, err := b.getFirstMessageInChannel(b.AppConfig.DiscordSrvSchedulerCid)
	if err != nil {
		log.Println("Discord bot failed to fetch notifications channel messages:", err)
		return
	}

	// found 1st channel message, editing it
	if firstMsg != nil {
		_, err = b.Session.ChannelMessageEdit(b.AppConfig.DiscordSrvSchedulerCid, firstMsg.ID, content)
		if err != nil {
			log.Println("Discord bot failed to edit notifications message:", err)
		}
		return
	}

	// initializing 1st channel message
	_, err = b.Session.ChannelMessageSend(b.AppConfig.DiscordSrvSchedulerCid, content)
	if err != nil {
		log.Println("Discord bot failed to send notifications message:", err)
	}
}

func (b *DiscordBot) dmUser(userID string, message string) error {
	channel, err := b.Session.UserChannelCreate(userID)
	if err != nil {
		return fmt.Errorf("failed to create DM channel with user %s: %w", userID, err)
	}

	// Send the message to the DM channel
	_, err = b.Session.ChannelMessageSend(channel.ID, message)
	if err != nil {
		return fmt.Errorf("failed to send DM to user %s: %w", userID, err)
	}

	return nil
}

func (b *DiscordBot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as: %s#%s\n", event.User.Username, event.User.Discriminator)
}

func (b *DiscordBot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	intent, err := b.IntentService.DetectIntent(m.ChannelID, m.Content)
	if err != nil {
		sentErrMsg, secondErr := s.ChannelMessageSend(m.ChannelID, err.Error())
		if secondErr != nil {
			log.Printf("Failed to send discord message: %s", err.Error())
		}
		b.tagMessageToBeDeleted(sentErrMsg, 180)
		return
	}
	if intent.Service == configs.ServiceNames.Scheduler {
		discordMetadata := &configs.DiscordMetadata{
			ChannelId: m.ChannelID,
			MessageId: m.MessageReference.MessageID,
			UserId:    m.Author.ID,
			Username:  m.Author.Username,
		}
		b.handleNotifications(intent, discordMetadata)
	}

	// switch m.Content {
	// case "!ping":
	// 	s.ChannelMessageSend(m.ChannelID, "Pong!")
	// 	s.ChannelMessageDelete(m.ChannelID, m.ID)
	// case "!hello":
	// 	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello, %s!", m.Author.Username))
	// 	s.ChannelMessageDelete(m.ChannelID, m.ID)
	// }
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
