package types

type EmailSender interface {
	Send(messageConfig MessageConfig) (any, error)
}

type MessageConfig struct {
	From    string
	To      []string
	Subject string
	Body    string
}

type StandardSenderConfig struct {
}
