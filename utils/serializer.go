package utils

import (
	"encoding/json"
	"fmt"
)

func StructToJson[T any](v T) (string, error) {
    b, err := json.Marshal(v)
    if err != nil {
        return "", fmt.Errorf("failed to marshal json string: %w", err)
    }
    return string(b), nil
}

func JsonToStruct[T any](data string) (T, error) {
    var v T
    if err := json.Unmarshal([]byte(data), &v); err != nil {
        return v, fmt.Errorf("failed to unmarshal struct: %w", err)
    }
    return v, nil
}
