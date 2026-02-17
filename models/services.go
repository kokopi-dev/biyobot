package models

import (
	"encoding/json"
	"fmt"
)

type ServiceResult struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`  // any valid JSON value
	Error string          `json:"error,omitempty"` // set only when ok=false
}

func Failure(msg string) ServiceResult {
	return ServiceResult{OK: false, Error: msg}
}

func Success(data any) ServiceResult {
	b, err := json.Marshal(data)
	if err != nil {
		return Failure(fmt.Sprintf("failed to marshal data: %v", err))
	}
	return ServiceResult{OK: true, Data: b}
}

func (r ServiceResult) Decode(target any) error {
	if !r.OK {
		return fmt.Errorf("service error: %s", r.Error)
	}
	return json.Unmarshal(r.Data, target)
}

type Runner interface {
	Run(input json.RawMessage) ServiceResult
}
