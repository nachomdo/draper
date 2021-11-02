# Kafka Distributed Tracing

This repository is derived from the work of Nacho Munoz and Samir Hafez as described in the blog post [Integrating Apache Kafka Clients with CNCF Jaeger at Funding Circle Using OpenTelemetry](https://www.confluent.io/blog/integrate-kafka-and-jaeger-for-distributed-tracing-and-monitoring/).

## Run in Gitpod

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/chuck-confluent/kafka-distributed-tracing)

## Prerequisites

Download OpenTelemetry Java Agent

```
wget -P agents https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v1.7.1/opentelemetry-javaagent-all.jar
```

Spin up docker compose stack 

```
docker-compose up -d
```

Deploy the datagen connectors, which produce Avro records to Kafka.

```
curl -X PUT -H "Content-type: application/json" -d @connectors/datagen-connector-trades.json http://localhost:8083/connectors/datagen-connector-trades/config

curl -X PUT -H "Content-type: application/json" -d @connectors/datagen-connector-users.json http://localhost:8083/connectors/datagen-connector-users/config
```

Open ksqlDB CLI prompt.

```bash
docker run --network kafka-distributed-tracing_default --rm --interactive --tty \
    -v ${PWD}/ksqldb_script.sql:/app/ksqldb_script.sql \
    confluentinc/ksqldb-cli:0.21.0 ksql \
    http://ksqldb-server:8088
```


Create streaming application in ksqlDB

```
ksql> SET 'auto.offset.reset' = 'earliest';

ksql> run script /app/ksqldb_script.sql
```

Try a push query

```
SELECT * FROM stockapp_dollars_by_zip_5_min EMIT CHANGES;
```

## View Metrics and Traces in Jaeger UI

Open http://localhost:16686 to see the Jaeger UI.