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

Create KSqlDB stream 

```
SET 'auto.offset.reset' = 'earliest';
 
CREATE STREAM stargazers_kafka with (KAFKA_TOPIC='github-avro-stargazers-kafka', VALUE_FORMAT='AVRO');

CREATE STREAM stargazers_jackdaw with (KAFKA_TOPIC='github-avro-stargazers-jackdaw', VALUE_FORMAT='AVRO');
```

Try a push query

```
SELECT data FROM stargazers_kafka EMIT CHANGES LIMIT 5;
```

Create topic backed stream to join users who starred Jackdaw and Kafka repositories 

```
CREATE STREAM stargazers_aggregate WITH (kafka_topic='stargazers-results', value_format='json', partitions='1') AS SELECT sgz.data->id as id, sgz.data->login as login, sgz.data->type as type FROM stargazers_kafka AS sgz INNER JOIN  stargazers_jackdaw as sjack WITHIN 10 days ON sgz.data->login=sjack.data->login  PARTITION BY sgz.data->id EMIT CHANGES;
```

### Troubleshoot

* Not getting any data out of KSqlDB queries

```
SET 'auto.offset.reset' = 'earliest';
```