package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ollama/ollama/api"
)

type IntentResult struct {
	Service    string         `json:"service"`
	Confidence float64        `json:"confidence"`
	Params     map[string]any `json:"params,omitempty"`
}

type ServiceConfig struct {
    KeywordsEN []string
    KeywordsJA []string
}

// manual definitions of services
var SERVICES = map[string]ServiceConfig{
    "currency_converter": {
        KeywordsEN: []string{"convert", "usd", "jpy", "yen", "dollar", "exchange"},
        KeywordsJA: []string{"両替", "変換", "円", "ドル", "いくら"},
    },
    "receipts": {
        KeywordsEN: []string{"receipt", "add", "scan", "expense"},
        KeywordsJA: []string{"レシート", "領収書", "追加", "経費"},
    },
    "scheduler": {
        KeywordsEN: []string{"schedule", "remind", "event", "meeting", "party"},
        KeywordsJA: []string{"スケジュール", "予定", "予約", "リマインド", "パーティー"},
    },
}


// return service name if keywords match
func keywordMatch(message string) string {
    msgLower := strings.ToLower(message)
    
    for serviceName, config := range SERVICES {
        for _, kw := range config.KeywordsEN {
            if strings.Contains(msgLower, strings.ToLower(kw)) {
                return serviceName
            }
        }
        for _, kw := range config.KeywordsJA {
            if strings.Contains(message, kw) {
                return serviceName
            }
        }
    }
    
    return ""
}

func extractParams(client *api.Client, service, message string) (map[string]any, error) {
    schemas := map[string]string{
        "currency_converter": `{"amount": number, "from_currency": "USD/JPY", "to_currency": "USD/JPY"}`,
        "receipts": `{"has_image": boolean}`,
        "schedule": `{"event_name": string, "date": "YYYY-MM-DD", "time": "HH:MM"}`,
    }

    prompt := fmt.Sprintf(`Extract parameters from this message for the %s service.
Message can be in English or Japanese.

Required parameters: %s

Message: "%s"

Return ONLY valid JSON matching the schema. Use current year 2026 if year is missing.`, 
        service, schemas[service], message)

    req := &api.ChatRequest{
        Model: "qwen2.5:3b",
        Messages: []api.Message{
            {Role: "user", Content: prompt},
        },
    }

    var fullResponse strings.Builder
    
    err := client.Chat(context.Background(), req, func(resp api.ChatResponse) error {
        fullResponse.WriteString(resp.Message.Content)
        return nil
    })

    if err != nil {
        return nil, err
    }

    jsonRegex := regexp.MustCompile(`\{[^}]+\}`)
    jsonStr := jsonRegex.FindString(fullResponse.String())
    
    var params map[string]any
    json.Unmarshal([]byte(jsonStr), &params)
    
    return params, nil
}

func DetectIntent(client *api.Client, message string) (*IntentResult, error) {
    // Try keyword match first
    service := keywordMatch(message)
    if service != "" {
        fmt.Printf("✓ Matched via keywords: %s\n", service)
        params, _ := extractParams(client, service, message)
        return &IntentResult{
            Service:    service,
            Confidence: 1.0,
            Params:     params,
        }, nil
    }
    
    // Fallback to LLM
    fmt.Println("⚠ No keyword match, using LLM...")
    
    prompt := fmt.Sprintf(`Detect which service the user wants. Message can be in English or Japanese.

Available services:
- currency_converter: convert money between currencies (両替、変換)
- receipts: scan and add receipt totals (レシート、領収書)
- schedule: schedule events or reminders (スケジュール、予定)

Message: "%s"

Return ONLY JSON: {"service": "service_name", "confidence": 0.95}
If unsure, set service to "unknown".`, message)

    req := &api.ChatRequest{
        Model: "qwen2.5:3b",
        Messages: []api.Message{
            {Role: "user", Content: prompt},
        },
    }

    var fullResponse strings.Builder
    
    err := client.Chat(context.Background(), req, func(resp api.ChatResponse) error {
        fullResponse.WriteString(resp.Message.Content)
        return nil
    })

    if err != nil {
        return nil, err
    }

    // Extract JSON from response
    jsonRegex := regexp.MustCompile(`\{[^}]+\}`)
    jsonStr := jsonRegex.FindString(fullResponse.String())
    
    var result IntentResult
    err = json.Unmarshal([]byte(jsonStr), &result)
    if err != nil {
        return &IntentResult{Service: "unknown", Confidence: 0.0}, nil
    }

    if result.Service != "unknown" && result.Confidence >= 0.7 {
        result.Params, _ = extractParams(client, result.Service, message)
    }

    return &result, nil
}
