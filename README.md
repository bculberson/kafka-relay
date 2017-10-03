# kafka-relay


To run:

```bash
docker-compose up
```

In one terminal (source):
```bash
docker-compose exec kafka /opt/kafka/bin/kafka-console-producer.sh --topic source --broker-list localhost:9092
```

In another terminal (destination):
```bash
docker-compose exec kafka /opt/kafka/binkafka-console-consumer.sh --topic destination --bootstrap-server localhost:9092
```

This will relay all messages you type into source terminal into destination.