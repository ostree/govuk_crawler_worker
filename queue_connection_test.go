package govuk_crawler_worker_test

import (
	. "github.com/alphagov/govuk_crawler_worker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/streadway/amqp"
)

var _ = Describe("QueueConnection", func() {
	It("fails if it can't connect to an AMQP server", func() {
		connection, err := NewQueueConnection("amqp://guest:guest@localhost:50000/")

		Expect(err).ToNot(BeNil())
		Expect(connection).To(BeNil())
	})

	Describe("Connecting to a running AMQP service", func() {
		var (
			connection    *QueueConnection
			connectionErr error
		)

		BeforeEach(func() {
			connection, connectionErr = NewQueueConnection("amqp://guest:guest@localhost:5672/")
		})

		AfterEach(func() {
			defer connection.Close()
		})

		It("successfully connects to an AMQP service", func() {
			Expect(connectionErr).To(BeNil())
			Expect(connection).ToNot(BeNil())
		})

		It("can close the connection without errors", func() {
			Expect(connection.Close()).To(BeNil())
		})

		It("can declare an exchange", func() {
			var err error
			exchange := "some-exchange"

			err = connection.ExchangeDeclare(exchange, "direct")
			Expect(err).To(BeNil())

			err = connection.Channel.ExchangeDelete(exchange, false, true)
			Expect(err).To(BeNil())
		})

		It("can declare a queue", func() {
			var (
				err   error
				queue amqp.Queue
				name  = "some-queue"
			)

			queue, err = connection.QueueDeclare(name)
			Expect(err).To(BeNil())
			Expect(queue.Name).To(Equal(name))

			deleted, err := connection.Channel.QueueDelete(name, false, false, true)
			Expect(err).To(BeNil())
			Expect(deleted).To(Equal(0))
		})

		It("can bind a queue to an exchange", func() {
			var err error

			exchangeName := "some-binding-exchange"
			queueName := "some-binding-queue"

			err = connection.ExchangeDeclare(exchangeName, "direct")
			Expect(err).To(BeNil())

			_, err = connection.QueueDeclare(queueName)
			Expect(err).To(BeNil())

			err = connection.BindQueueToExchange(queueName, exchangeName)
			Expect(err).To(BeNil())

			deleted, err := connection.Channel.QueueDelete(queueName, false, false, true)
			Expect(err).To(BeNil())
			Expect(deleted).To(Equal(0))

			err = connection.Channel.ExchangeDelete(exchangeName, false, true)
			Expect(err).To(BeNil())
		})
	})

	Describe("working with messages on the queue", func() {
		var (
			publisher *QueueConnection
			consumer  *QueueConnection
			err       error
		)

		exchangeName := "test-crawler-exchange"
		queueName := "test-crawler-queue"

		BeforeEach(func() {
			publisher, err = NewQueueConnection("amqp://guest:guest@localhost:5672/")
			Expect(err).To(BeNil())
			Expect(publisher).ToNot(BeNil())

			consumer, err = NewQueueConnection("amqp://guest:guest@localhost:5672/")
			Expect(err).To(BeNil())
			Expect(consumer).ToNot(BeNil())
		})

		AfterEach(func() {
			deleted, err := consumer.Channel.QueueDelete(queueName, false, false, true)
			Expect(err).To(BeNil())
			Expect(deleted).To(Equal(0))

			err = consumer.Channel.ExchangeDelete(exchangeName, false, true)
			Expect(err).To(BeNil())

			Expect(publisher.Close()).To(BeNil())
			Expect(consumer.Close()).To(BeNil())
		})

		It("should consume and publish messages onto the provided queue and exchange", func() {
			err = consumer.ExchangeDeclare(exchangeName, "direct")
			Expect(err).To(BeNil())

			_, err = consumer.QueueDeclare(queueName)
			Expect(err).To(BeNil())

			err = consumer.BindQueueToExchange(queueName, exchangeName)
			Expect(err).To(BeNil())

			deliveries, err := consumer.Consume(queueName)
			Expect(err).To(BeNil())

			err = publisher.Publish(exchangeName, "#", "text/plain", "foo")
			Expect(err).To(BeNil())

			for d := range deliveries {
				Expect(string(d.Body)).To(Equal("foo"))
				break
			}
		})
	})
})