package main

import (
	"github.com/alphagov/govuk_crawler_worker/queue"
	"github.com/alphagov/govuk_crawler_worker/ttl_hash_set"
)

type Status struct {
	AMQP  bool `json:"amqp"`
	Redis bool `json:"redis"`
}

type HealthCheck struct {
	port         string
	queueManager *queue.QueueManager
	ttlHashSet   *ttl_hash_set.TTLHashSet
}

func NewHealthCheck(queueManager *queue.QueueManager, ttlHashSet *ttl_hash_set.TTLHashSet) *HealthCheck {
	return &HealthCheck{
		queueManager: queueManager,
		ttlHashSet:   ttlHashSet,
	}
}

func (h *HealthCheck) Status() *Status {
	var consumerStatus, publisherStatus, redisStatus bool

	pong, err := h.ttlHashSet.Ping()
	if err == nil && pong == "PONG" {
		redisStatus = true
	}

	consumerInspect, err := h.queueManager.Consumer.Channel.QueueInspect(h.queueManager.QueueName)
	if err == nil && consumerInspect.Name == h.queueManager.QueueName {
		consumerStatus = true
	}

	publisherInspect, err := h.queueManager.Producer.Channel.QueueInspect(h.queueManager.QueueName)
	if err == nil && publisherInspect.Name == h.queueManager.QueueName {
		publisherStatus = true
	}

	return &Status{
		AMQP:  (consumerStatus && publisherStatus),
		Redis: redisStatus,
	}
}
