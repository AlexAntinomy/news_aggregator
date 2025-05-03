package queue

import (
	"context"
	"news_aggregator/internal/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Producer
type Producer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewProducer(url string) (*Producer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Producer{conn, ch}, nil
}

func (p *Producer) Publish(queueName string, body []byte) error {
	// Явно объявляем очередь с durable=true
	_, err := p.ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(
		context.Background(),
		"",        // exchange
		queueName, // routing key (имя очереди)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Сохранять сообщения при перезапуске
			ContentType:  "text/plain",
			Body:         body,
		},
	)
}

func (p *Producer) Close() {
	p.ch.Close()
	p.conn.Close()
}

// Consumer
type Consumer struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	queue   string
	workers int
}

func NewConsumer(url, queue string, workers int) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Consumer{
		conn:    conn,
		ch:      ch,
		queue:   queue,
		workers: workers,
	}, nil
}

func (c *Consumer) Consume(handler func([]byte) error) {
	// Объявляем очередь с теми же параметрами, что и Producer
	q, err := c.ch.QueueDeclare(
		c.queue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		logger.Log.Errorf("Queue declare error: %v", err)
		return
	}

	logger.Log.Infof("Consuming queue: %s (messages: %d)", q.Name, q.Messages)

	msgs, err := c.ch.Consume(
		q.Name,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		logger.Log.Errorf("Consume failed: %v", err)
		return
	}

	for i := 0; i < c.workers; i++ {
		go func() {
			for msg := range msgs {
				if err := handler(msg.Body); err == nil {
					msg.Ack(false)
				} else {
					msg.Nack(false, true)
					logger.Log.Errorf("Task failed: %v", err)
				}
			}
		}()
	}
}

func (c *Consumer) Close() {
	c.ch.Close()
	c.conn.Close()
}
