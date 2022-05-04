package schemas

import (
	"encoding/json"
)

type GuiderSchema struct {
	Schema *AISchema `json:"schema"`
}

func LoadGuiderSchema(b []byte) error {
	gs := GuiderSchema{}
	if err := json.Unmarshal(b, &gs); err != nil {
		return err
	}
	return LoadSchema(gs.Schema)
}

type EngineSchema struct {
	Schema *AISchema `json:"schema"`
}

func LoadEngineSchema(b []byte) error {
	gs := EngineSchema{}
	if err := json.Unmarshal(b, &gs); err != nil {
		return err
	}

	return LoadSchema(gs.Schema)
}
