package config

import "github.com/microcosm-cc/bluemonday"

type BackendConfig struct {
	ListenPort             string
	FrontendEndpoint       string
	BackendEndpoint        string
	HTMLSanitizationPolicy *bluemonday.Policy
}
