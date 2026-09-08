package publisher

import (
	"context"

	enc_module "agreements-generator/internal/encoder"
	"agreements-generator/internal/publisher/rabbit_publisher"
)

type Publisher interface {
	Send(ctx context.Context, jobID string) error
	Close() error
}

func New(
	host string,
	port int,
	username,
	password,
	vhost,
	queue string,
	encoder enc_module.Encoder,
) (Publisher, error) {
	return rabbit_publisher.New(host, port, username, password, vhost, queue, encoder)
}
