package publisher

import (
	"context"

	"agreements-generator/internal/encoder"
	"agreements-generator/internal/publisher/rabbit_publisher"
)

type Publisher interface {
	Send(ctx context.Context, jobID string) error
	Close() error
}

func New(
	host string,
	port int,
	username string,
	password string,
	vhost string,
	queue string,
	encoder encoder.Encoder,
) (Publisher, error) {
	return rabbit_publisher.New(host, port, username, password, vhost, queue, encoder)
}
