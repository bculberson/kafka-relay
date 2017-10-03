package main

import (
	"os"
	"os/signal"
	"strings"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
)

const group = "relayConsumer"

func main() {
	sourceBroker := strings.Split(os.Getenv("SOURCE"), ",")
	sourceTopic := os.Getenv("SOURCE_TOPIC")
	destBroker := strings.Split(os.Getenv("DESTINATION"), ",")
	destTopic := os.Getenv("DESTINATION_TOPIC")

	consumerConfig := cluster.NewConfig()
	consumer, err := cluster.NewConsumer(sourceBroker, group, []string{sourceTopic}, consumerConfig)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(destBroker, producerConfig)
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	for {
		select {
		case msg, ok := <-consumer.Messages():
			if ok {
				newMsg := &sarama.ProducerMessage{Topic: destTopic, Value: sarama.ByteEncoder(msg.Value), Partition: 1}
				_, _, err := producer.SendMessage(newMsg)
				if err != nil {
					panic(err)
				}
				consumer.MarkOffset(msg, "") // mark message as processed
			}
		case <-signals:
			return
		}
	}

}
