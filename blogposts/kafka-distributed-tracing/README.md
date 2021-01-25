![image](https://user-images.githubusercontent.com/3109377/105387384-642e0580-5c0d-11eb-9365-3ce778d42466.png)


## Distributed tracing and Kafka

### Getting started

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