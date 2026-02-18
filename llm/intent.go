package llm

import (
	"biyobot/services/database"
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
	Actions    []Action
	KeywordsEN []string
	KeywordsJA []string
}

type Action struct {
	Name       string
	KeywordsEN []string
	KeywordsJA []string
	Schema     map[string]string
}

type IntentResult struct {
	Service    string         `json:"service"`
	Action     string         `json:"action"`
	Confidence float64        `json:"confidence"`
	Params     map[string]any `json:"params,omitempty"`
}

// manual definitions of services
var SERVICES = map[string]Service{
	"scheduler": {
		KeywordsEN: []string{"schedule", "event", "meeting", "party", "appointment"},
		KeywordsJA: []string{"スケジュール", "予定", "予約", "イベント"},
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
					"notification_id":   "string (required)",
					"notify_at":   "RFC3339 (2006-01-02T15:04:05Z07:00) (required)",
					"title":       "string (required)",
					"description": "string (required)",
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
		KeywordsEN: []string{"receipt", "expense", "scan"},
		KeywordsJA: []string{"レシート", "領収書", "経費"},
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
		KeywordsEN: []string{"convert", "exchange", "currency"},
		KeywordsJA: []string{"両替", "変換", "換算"},
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

func getJapanTimeNow() (time.Time) {
	now := time.Now().In(time.FixedZone("JST", 9*60*60)) // UTC+9
	return now
}

func keywordMatchService(message string) string {
	msgLower := strings.ToLower(message)

	for serviceName, service := range SERVICES {
		for _, kw := range service.KeywordsEN {
			if strings.Contains(msgLower, strings.ToLower(kw)) {
				return serviceName
			}
		}
		for _, kw := range service.KeywordsJA {
			if strings.Contains(message, kw) {
				return serviceName
			}
		}
	}

	return ""
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

func DetectIntent(client *api.Client, message string, notificationRepo *database.NotificationsRepo) (*IntentResult, error) {
	// Step 1: Match service via keywords
	serviceName := keywordMatchService(message)

	var service Service
	var usingLLM bool

	if serviceName == "" {
		// LLM fallback for service
		serviceName, usingLLM = llmDetectService(client, message)
		if serviceName == "unknown" {
			return &IntentResult{Service: "unknown", Confidence: 0.0}, nil
		}
	}

	service = SERVICES[serviceName]

	// Step 2: Match action via keywords
	actionName := keywordMatchAction(service, message)

	if actionName == "" {
		// LLM fallback for action
		actionName = llmDetectAction(client, serviceName, service, message, notificationRepo)
		usingLLM = true
	}

	// Default to first action if still unknown
	if actionName == "" && len(service.Actions) > 0 {
		actionName = service.Actions[0].Name
	}

	// Step 3: Extract params for this action
	var action Action
	for _, a := range service.Actions {
		if a.Name == actionName {
			action = a
			break
		}
	}

	params := extractParams(client, serviceName, actionName, action.Schema, message, notificationRepo)

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

func llmDetectService(client *api.Client, message string) (string, bool) {
	var serviceList strings.Builder
	for name, service := range SERVICES {
		enKw := strings.Join(service.KeywordsEN[:min(3, len(service.KeywordsEN))], ", ")
		jaKw := ""
		if len(service.KeywordsJA) > 0 {
			jaKw = " / " + strings.Join(service.KeywordsJA[:min(3, len(service.KeywordsJA))], ", ")
		}
		fmt.Fprintf(&serviceList, "- %s: %s%s\n", name, enKw, jaKw)
	}

	prompt := fmt.Sprintf(`Detect which service the user wants.

Available services:
%s
Message: "%s"

Return ONLY JSON: {"service": "service_name", "confidence": 0.95}`, serviceList.String(), message)

	response := callLLM(client, prompt)

	var result struct {
		Service    string  `json:"service"`
		Confidence float64 `json:"confidence"`
	}

	jsonRegex := regexp.MustCompile(`\{[^}]+\}`)
	jsonStr := jsonRegex.FindString(response)
	json.Unmarshal([]byte(jsonStr), &result)

	return result.Service, true
}

func buildContext(serviceName string, notificationRepo *database.NotificationsRepo) string {
	var contextStr string
	if serviceName == "scheduler" {
        notifications, err := notificationRepo.GetAllNotifications()
        if err == nil && len(notifications) > 0 {
            var notifList strings.Builder
            notifList.WriteString("Existing events:\n")
            for _, n := range notifications {
                fmt.Fprintf(&notifList, "- ID:%d, Name:\"%s\", Time:%s\n", 
                    n.ID, n.Message, n.NotifyAt.Format(time.RFC3339))
            }
            contextStr = notifList.String()
        }
    }
	return contextStr
}

func llmDetectAction(client *api.Client, serviceName string, service Service, message string, notificationRepo *database.NotificationsRepo) string {
	contextStr := buildContext(serviceName, notificationRepo)
	var actionList strings.Builder
	for _, action := range service.Actions {
		enKw := strings.Join(action.KeywordsEN[:min(3, len(action.KeywordsEN))], ", ")
		jaKw := ""
		if len(action.KeywordsJA) > 0 {
			jaKw = " / " + strings.Join(action.KeywordsJA[:min(3, len(action.KeywordsJA))], ", ")
		}
		fmt.Fprintf(&actionList, "- %s: %s%s\n", action.Name, enKw, jaKw)
	}

	now := getJapanTimeNow()
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

	response := callLLM(client, prompt)

	var result struct {
		Action string `json:"action"`
	}

	jsonRegex := regexp.MustCompile(`\{[^}]+\}`)
	jsonStr := jsonRegex.FindString(response)
	json.Unmarshal([]byte(jsonStr), &result)

	return result.Action
}

func extractParams(client *api.Client, serviceName string, actionName string, schema map[string]string, message string, notificationRepo *database.NotificationsRepo) map[string]any {
	now := getJapanTimeNow()
	contextStr := buildContext(serviceName, notificationRepo)
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

	response := callLLM(client, prompt)

	jsonRegex := regexp.MustCompile(`\{[^}]+\}`)
	jsonStr := jsonRegex.FindString(response)

	var params map[string]any
	json.Unmarshal([]byte(jsonStr), &params)

	return params
}

func callLLM(client *api.Client, prompt string) string {
	log.Println("Using LLM...")
	req := &api.ChatRequest{
		Model: "qwen2.5:3b",
		Messages: []api.Message{
			{Role: "user", Content: prompt},
		},
	}

	var fullResponse strings.Builder
	client.Chat(context.Background(), req, func(resp api.ChatResponse) error {
		fullResponse.WriteString(resp.Message.Content)
		return nil
	})

	return fullResponse.String()
}
