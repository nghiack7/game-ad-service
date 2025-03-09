package messaging

// MessageBroker interface defines the methods for a message broker
type MessageBroker interface {
	Publish(topic string, message []byte) error
	Subscribe(topic string, handler func(message []byte)) error
}

// MessageHandler is a function that handles a message
type MessageHandler func(message []byte) error

// MessagePublisher is a function that publishes a message
type MessagePublisher func(topic string, message []byte) error

// MessageSubscriber is a function that subscribes to a topic
type MessageSubscriber func(topic string, handler MessageHandler) error
