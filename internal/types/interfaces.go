package types

type EventPublisher interface {
	PublishNewFileLoadedEvent() error
}