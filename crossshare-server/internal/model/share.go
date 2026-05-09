package model

type Share struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Content     []byte      `json:"-"`
	ContentType string      `json:"content_type"`
	ContentSize int         `json:"content_size"`
	StoredSize  int         `json:"stored_size"`
	Hash        string      `json:"hash"`
	CreatedAt   int64       `json:"created_at"`
	ExpireAt    int64       `json:"expire_at"`
	Creator     string      `json:"creator"`
	Type        string      `json:"type"`
	Files       []ShareFile `json:"files,omitempty"`
}

type ShareFile struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Hash        string `json:"hash"`
}
