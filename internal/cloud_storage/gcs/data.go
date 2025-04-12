package gcs

import "time"

type BucketData struct {
	BucketName   string            `json:"bucket_name"`
	StorageClass string            `json:"storage_class"`
	TimeCreated  time.Time         `json:"time_created"`
	Labels       map[string]string `json:"labels"`
	Objects      []ObjectMedatata  `json:"objects"`
}

type ObjectMedatata struct {
	ContentType string    `json:"content_type"`
	Created     time.Time `json:"date_created"`
	Deleted     time.Time `json:"date_deleted"`
	Updated     time.Time `json:"date_updated"`
	MD5         []byte    `json:"md5"`
	Size        int64     `json:"size"`
	MediaLink   string    `json:"media_link"`
}
