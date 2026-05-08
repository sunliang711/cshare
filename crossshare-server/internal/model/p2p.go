package model

import "encoding/json"

type P2PSessionResult struct {
	SessionID string `json:"session_id"`
	TTL       int    `json:"ttl"`
	ExpireAt  int64  `json:"expire_at"`
}

type P2PMessage struct {
	Seq     int64           `json:"seq"`
	From    string          `json:"from"`
	To      string          `json:"to"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type P2PMessagesResult struct {
	Messages []P2PMessage `json:"messages"`
}
