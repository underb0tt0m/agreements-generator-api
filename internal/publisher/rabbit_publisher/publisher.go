package rabbit_publisher

import (
	"context"
	"fmt"

	"agreements-generator/internal/encoder"

	amqp "github.com/rabbitmq/amqp091-go"
)

type DeliveryJob struct {
	JobID string `json:"job_id"`
}

type rabbitPublisher struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	queueName string
	encoder   encoder.Encoder
}

func New(
	host string,
	port int,
	username string,
	password string,
	vhost string,
	queue string,
	encoder encoder.Encoder,
) (*rabbitPublisher, error) { // TODO подумать на циклическими импортами
	connString := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		username, password, host, port, vhost,
	)

	conn, err := amqp.Dial(connString)
	if err != nil {
		return nil, fmt.Errorf("can't connect to rabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("can't create channel: %w", err)
	}
	_, err = ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("can't declare queue: %w", err)
	}

	return &rabbitPublisher{
		conn:      conn,
		ch:        ch,
		queueName: queue,
		encoder:   encoder,
	}, nil
}

func (p *rabbitPublisher) Send(ctx context.Context, jobID string) error {
	delivery, err := p.newPublishing(DeliveryJob{JobID: jobID})
	if err != nil {
		return err
	}

	if err = p.ch.PublishWithContext(
		ctx,
		"",
		p.queueName,
		false,
		false,
		*delivery,
	); err != nil {
		return fmt.Errorf("can't send job to queue: %w", err)
	}

	return nil
}

func (p *rabbitPublisher) Close() error {
	return p.conn.Close()
}

func (p *rabbitPublisher) newPublishing(obj any) (*amqp.Publishing, error) {
	body, err := p.encoder.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("can't marshal object for delivering: %w", err)
	}
	return &amqp.Publishing{
		Type:         "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}, nil
}
