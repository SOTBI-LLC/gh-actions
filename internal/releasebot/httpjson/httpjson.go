package httpjson

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultMaxBodyBytes = 1 << 20

func DecodeBody(body io.Reader, target any, disallowUnknownFields ...bool) error {
	decoder := json.NewDecoder(io.LimitReader(body, defaultMaxBodyBytes))
	if len(disallowUnknownFields) == 0 || disallowUnknownFields[0] {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}

	return nil
}

func Write(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
