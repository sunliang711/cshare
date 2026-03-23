package model

type Response struct {
	Code      int         `json:"code"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

type PushResult struct {
	Key      string `json:"key"`
	TTL      int    `json:"ttl"`
	Size     int    `json:"size"`
	Type     string `json:"type"`
	Filename string `json:"filename,omitempty"`
	ExpireAt int64  `json:"expire_at"`
}

type PullTextResult struct {
	Key         string `json:"key"`
	Text        string `json:"text"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Deleted     bool   `json:"deleted"`
}

type DeleteResult struct {
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

type HealthResult struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Time    string `json:"time"`
}
