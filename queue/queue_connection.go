package queue

import (
	"log"

	"github.com/streadway/amqp"
)

type QueueConnection struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel

	url         string
	notifyClose chan *amqp.Error
}

func NewQueueConnection(amqpURI string) (*QueueConnection, error) {
	queueConnection := &QueueConnection{
		url: amqpURI,
	}
	err := queueConnection.Connect()
	if err != nil {
		return nil, err
	}

	return queueConnection, nil
}

func (c *QueueConnection) Connect() error {
	connection, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}

	channel, err := connection.Channel()
	if err != nil {
		return err
	}

	err = channel.Qos(5, 0, false)
	if err != nil {
		return err
	}

	c.Connection = connection
	c.Channel = channel
	c.notifyClose = c.Channel.NotifyClose(make(chan *amqp.Error))

	go func() {
		select {
		case e := <-c.notifyClose:
			if e == amqp.ErrClosed {
				log.Fatal(e)
			}
		}
	}()

	return nil
}

func (c *QueueConnection) Close() error {
	err := c.Channel.Close()
	if err != nil {
		return err
	}

	return c.Connection.Close()
}

func (c *QueueConnection) Consume(queueName string) (<-chan amqp.Delivery, error) {
	return c.Channel.Consume(
		queueName,
		"",
		false, // autoAck
		false, // this won't be the sole consumer
		true,  // don't deliver messages from same connection
		false, // the broker owns when consumption can begin
		nil)   // arguments
}

func (c *QueueConnection) ExchangeDeclare(exchangeName string, exchangeType string) error {
	return c.Channel.ExchangeDeclare(
		exchangeName, // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	)
}

func (c *QueueConnection) QueueDeclare(queueName string) (amqp.Queue, error) {
	queue, err := c.Channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when usused
		false,     // exclusive
		false,     // noWait
		nil)       // arguments
	if err != nil {
		return amqp.Queue{
			Name: queueName,
		}, err
	}

	return queue, nil
}

func (c *QueueConnection) BindQueueToExchange(queueName string, exchangeName string) error {
	return c.Channel.QueueBind(
		queueName,
		"#", // key to marshall with
		exchangeName,
		true, // noWait
		nil)  // arguments
}

func (c *QueueConnection) Publish(exchangeName string, routingKey string, contentType string, body string) error {
	return c.Channel.Publish(
		exchangeName, // publish to an exchange
		routingKey,   // routing to 0 or more queues
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     contentType,
			ContentEncoding: "",
			Body:            []byte(body),
			DeliveryMode:    amqp.Persistent,
			Priority:        0, // 0-9
		})
}
