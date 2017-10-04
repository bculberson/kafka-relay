package main

import (
	"bytes"
	"encoding/json"
	"encoding/binary"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
)

const group = "relayConsumer"

func main() {
	sourceBroker := strings.Split(os.Getenv("SOURCE"), ",")
	sourceTopic := os.Getenv("SOURCE_TOPIC")
	sourceKeyField := os.Getenv("SOURCE_KEY_FIELD")
	destBroker := strings.Split(os.Getenv("DESTINATION"), ",")
	destTopic := os.Getenv("DESTINATION_TOPIC")
	destPartitionsEnv := os.Getenv("DESTINATION_PARTITIONS")
	destPartitions, err := strconv.Atoi(destPartitionsEnv)
	if err != nil {
		panic(err)
	}

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
	producerConfig.Producer.Compression = sarama.CompressionSnappy
	producer, err := sarama.NewSyncProducer(destBroker, producerConfig)
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	partitioner := sarama.NewHashPartitioner(destTopic)

	for {
		select {
		case msg, ok := <-consumer.Messages():
			if ok {
				value := make(map[string]interface{})
				err := json.Unmarshal(msg.Value, &value)
				if err != nil {
					log.Printf("Error unmarshalling json")
					continue
				}
				var key interface{}
				if key, ok = value[sourceKeyField]; !ok {
					log.Printf("Error finding key %s from json", sourceKeyField)
					continue
				}

				var newMsg *sarama.ProducerMessage
				if keyAsString, ok := key.(string); ok {
					newMsg = &sarama.ProducerMessage{Topic: destTopic, Value: sarama.ByteEncoder(msg.Value), Key: sarama.StringEncoder(keyAsString)}
				} else {
					buf := new(bytes.Buffer)
					binary.Write(buf, binary.LittleEndian, key)
					newMsg = &sarama.ProducerMessage{Topic: destTopic, Value: sarama.ByteEncoder(msg.Value), Key: sarama.ByteEncoder(buf.Bytes())}
				}

				partition, err := partitioner.Partition(newMsg, int32(destPartitions))
				if err != nil {
					panic(err)
				}
				newMsg.Partition = partition
				_, _, err = producer.SendMessage(newMsg)
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
