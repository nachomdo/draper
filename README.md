# Kafka Distributed Tracing

This repository is derived from the work of Nacho Munoz and Samir Hafez as described in the blog post [Integrating Apache Kafka Clients with CNCF Jaeger at Funding Circle Using OpenTelemetry](https://www.confluent.io/blog/integrate-kafka-and-jaeger-for-distributed-tracing-and-monitoring/).

## Run in Gitpod

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/chuck-confluent/kafka-distributed-tracing)

## Instructions

Download OpenTelemetry Java Agent

```
wget -P agents https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v1.7.1/opentelemetry-javaagent-all.jar
```


Install Github source connector. 

```
mkdir confluent-hub-components

confluent-hub install --component-dir ./confluent-hub-components  --no-prompt confluentinc/kafka-connect-github:latest  
```

Spin up docker compose stack 

```
docker-compose up -d
```

Edit the connectors to update GH Token in the configuration and deploy the connectors to start sourcing data

```
curl -X PUT -H "Content-type: application/json" -d @connectors/gh-connector-kafka.json http://localhost:8083/connectors/gh-connector-avro-kafka/config

curl -X PUT -H "Content-type: application/json" -d @connectors/gh-connector-jackdaw.json http://localhost:8083/connectors/gh-connector-avro-jackdaw/config
```

Create KSqlDB streams 

```
ksql> SET 'auto.offset.reset' = 'earliest';
 
ksql> run script ./ksqldb_script.sql
```

Try a push query

```
SELECT data FROM stargazers_kafka EMIT CHANGES LIMIT 5;
```

### Troubleshoot

* Not getting any data out of KSqlDB queries

```
SET 'auto.offset.reset' = 'earliest';
```