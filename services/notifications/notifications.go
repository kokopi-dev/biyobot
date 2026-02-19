package notifications

import (
	"biyobot/configs"
	"encoding/json"
	"time"
)

type Input struct {
	Action   string    `json:"action"`
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	NotifyAt time.Time `json:"notify_at"`
}

type Output struct {
	ResultMessage string `json:"message"`
}

type Service struct{}

func (s *Service) Run(msg json.RawMessage) configs.ServiceResult {
	var input Input
	if err := json.Unmarshal(msg, &input); err != nil {
		return configs.Failure("invalid input: " + err.Error())
	}

	service := "discord" // hardcoded discord
	metadata := map[string]any{} // hardcoded discord metadata: userid, channelid

	if input.Action == "" {
		return configs.Failure("`action` is required")
	}

	switch input.Action {
		case "add":
			if input.Message == "" {
				return configs.Failure("`message` is required")
			}
			if input.Title == "" {
				return configs.Failure("`message` is required")
			}
		case "edit":
			if input.ID == "" {
				return configs.Failure("`id` is required")
			}
		case "delete":
			if input.ID == "" {
				return configs.Failure("`id` is required")
			}
		default:
			return configs.Failure("`action` can only be `add | edit | delete`")
	}

	return configs.Success(Output{
		ResultMessage: "",
	})
}
