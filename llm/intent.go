package llm

import (
	"biyobot/configs"
	"biyobot/services/database"
	"biyobot/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

type Service struct {
	DiscordChannelID string
	Actions          []Action
	KeywordsEN       []string
	KeywordsJA       []string
}

type Action struct {
	Name       string
	KeywordsEN []string
	KeywordsJA []string
	Schema     map[string]string
}

type IntentResult struct {
	Service string `json:"service"`
	// add | edit | delete
	Action     string         `json:"action"`
	Confidence float64        `json:"confidence"`
	Params     map[string]any `json:"params,omitempty"`
}

type IntentService struct {
	client           *api.Client
	notificationRepo *database.NotificationsRepo
	services         map[string]Service
	channelIdx       map[string]string
}

func NewIntentService(client *api.Client, notificationRepo *database.NotificationsRepo, appConfig *configs.AppConfig) *IntentService {
	services := map[string]Service{
		configs.ServiceNames.Scheduler: {
			DiscordChannelID: appConfig.DiscordSrvSchedulerCid,
			KeywordsEN:       []string{"schedule", "event", "meeting", "party", "appointment"},
			KeywordsJA:       []string{"スケジュール", "予定", "予約", "イベント"},
			Actions: []Action{
				{
					Name:       "add",
					KeywordsEN: []string{"add", "create", "schedule", "set", "new"},
					KeywordsJA: []string{"追加", "作成", "入れる", "設定"},
					Schema: map[string]string{
						"notify_at":   "RFC3339 (2006-01-02T15:04:05Z07:00) (required)",
						"title":       "string (required)",
						"description": "string (required)",
					},
				},
				{
					Name:       "edit",
					KeywordsEN: []string{"edit", "update", "change", "modify", "reschedule"},
					KeywordsJA: []string{"編集", "変更", "修正", "更新"},
					Schema: map[string]string{
						"notification_id": "string (required)",
						"notify_at":       "RFC3339 (2006-01-02T15:04:05Z07:00) (required)",
						"title":           "string (required)",
						"description":     "string (required)",
					},
				},
				{
					Name:       "delete",
					KeywordsEN: []string{"delete", "remove", "cancel"},
					KeywordsJA: []string{"削除", "消去", "キャンセル"},
					Schema: map[string]string{
						"notification_id": "string (required)",
					},
				},
			},
		},
		"receipts": {
			DiscordChannelID: "",
			KeywordsEN:       []string{"receipt", "expense", "scan"},
			KeywordsJA:       []string{"レシート", "領収書", "経費"},
			Actions: []Action{
				{
					Name:       "add",
					KeywordsEN: []string{"add", "scan", "log"},
					KeywordsJA: []string{"追加", "スキャン", "記録"},
					Schema: map[string]string{
						"has_image": "boolean",
					},
				},
			},
		},
		"currency_converter": {
			DiscordChannelID: "",
			KeywordsEN:       []string{"convert", "exchange", "currency"},
			KeywordsJA:       []string{"両替", "変換", "換算"},
			Actions: []Action{
				{
					Name:       "convert",
					KeywordsEN: []string{"convert", "to", "exchange"},
					KeywordsJA: []string{"変換", "換算", "両替"},
					Schema: map[string]string{
						"amount":        "number",
						"from_currency": "string",
						"to_currency":   "string",
					},
				},
			},
		},
	}

	channelIndex := make(map[string]string, len(services))
	for name, svc := range services {
		if svc.DiscordChannelID != "" {
			channelIndex[svc.DiscordChannelID] = name
		}
	}

	return &IntentService{
		client:           client,
		notificationRepo: notificationRepo,
		services:         services,
		channelIdx:       channelIndex,
	}
}

func (s *IntentService) DetectIntent(channelID, message string) (*IntentResult, error) {
	serviceName, ok := s.channelIdx[channelID]
	if !ok {
		return &IntentResult{Service: "unknown", Confidence: 0.0}, nil
	}

	service := s.services[serviceName]

	var usingLLM bool

	actionName := keywordMatchAction(service, message)
	if actionName == "" {
		actionName = s.llmDetectAction(serviceName, service, message)
		usingLLM = true
	}

	if actionName == "" && len(service.Actions) > 0 {
		actionName = service.Actions[0].Name
	}

	var action Action
	for _, a := range service.Actions {
		if a.Name == actionName {
			action = a
			break
		}
	}

	params := s.extractParams(serviceName, actionName, action.Schema, message)

	confidence := 1.0
	if usingLLM {
		confidence = 0.85
	}

	return &IntentResult{
		Service:    serviceName,
		Action:     actionName,
		Params:     params,
		Confidence: confidence,
	}, nil
}

func (s *IntentService) llmDetectAction(serviceName string, service Service, message string) string {
	contextStr := s.buildContext(serviceName)

	var actionList strings.Builder
	for _, action := range service.Actions {
		enKw := strings.Join(action.KeywordsEN[:min(3, len(action.KeywordsEN))], ", ")
		jaKw := ""
		if len(action.KeywordsJA) > 0 {
			jaKw = " / " + strings.Join(action.KeywordsJA[:min(3, len(action.KeywordsJA))], ", ")
		}
		fmt.Fprintf(&actionList, "- %s: %s%s\n", action.Name, enKw, jaKw)
	}

	now := utils.JapanTimeNow()
	prompt := fmt.Sprintf(`Detect which action the user wants for the %s service.

Context:
- Current Time: %s
- Current Data: %s

Available actions:
%s

Message: "%s"

Rules:
- If user mentions an existing event by name or ID, likely "edit" or "delete"
- If user says "add", "create", "new", or describes a new event, likely "add"
- If context is unclear, default to "add"

Return ONLY JSON: {"action": "action_name"}`, serviceName, now.Format(time.RFC3339), contextStr, actionList.String(), message)

	response := s.callLLM(prompt)

	var result struct {
		Action string `json:"action"`
	}
	if jsonStr := extractJSON(response); jsonStr != "" {
		json.Unmarshal([]byte(jsonStr), &result)
	}

	return result.Action
}

func (s *IntentService) extractParams(serviceName, actionName string, schema map[string]string, message string) map[string]any {
	now := utils.JapanTimeNow()
	contextStr := s.buildContext(serviceName)
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")

	prompt := fmt.Sprintf(`Extract parameters from this message for the %s.%s action.

Context:
- Current Time: %s
- Current Data: %s

Required schema:
%s

Message: "%s"

Rules:
- For edit/delete: If user mentions an event by name, find the matching ID from the list above.
- For datetime: Use RFC3339 format with +09:00 timezone.

Return ONLY valid JSON matching the schema. Use 2026 for missing years.`,
		serviceName, actionName, now.Format(time.RFC3339), contextStr, string(schemaJSON), message)

	response := s.callLLM(prompt)

	var params map[string]any
	if jsonStr := extractJSON(response); jsonStr != "" {
		json.Unmarshal([]byte(jsonStr), &params)
	}

	return params
}

func (s *IntentService) buildContext(serviceName string) string {
	if serviceName != "scheduler" {
		return ""
	}

	notifications, err := s.notificationRepo.GetAllNotifications()
	if err != nil || len(notifications) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Existing events:\n")
	for _, n := range notifications {
		fmt.Fprintf(&b, "- ID:%d, Name:\"%s\", Time:%s\n", n.ID, n.Message, n.NotifyAt.Format(time.RFC3339))
	}
	return b.String()
}

func (s *IntentService) callLLM(prompt string) string {
	log.Println("Using LLM...")
	req := &api.ChatRequest{
		Model: "qwen2.5:3b",
		Messages: []api.Message{
			{Role: "user", Content: prompt},
		},
	}

	var b strings.Builder
	s.client.Chat(context.Background(), req, func(resp api.ChatResponse) error {
		b.WriteString(resp.Message.Content)
		return nil
	})

	return b.String()
}

func extractJSON(s string) string {
	return regexp.MustCompile(`\{[^}]+\}`).FindString(s)
}

func keywordMatchAction(service Service, message string) string {
	msgLower := strings.ToLower(message)
	for _, action := range service.Actions {
		for _, kw := range action.KeywordsEN {
			if strings.Contains(msgLower, strings.ToLower(kw)) {
				return action.Name
			}
		}
		for _, kw := range action.KeywordsJA {
			if strings.Contains(message, kw) {
				return action.Name
			}
		}
	}
	return ""
}
