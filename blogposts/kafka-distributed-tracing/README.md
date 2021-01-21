![image](https://user-images.githubusercontent.com/3109377/105387384-642e0580-5c0d-11eb-9365-3ce778d42466.png)


## Distributed tracing and Kafka

### Getting started

Install Github and S3 source connectors. 

```
mkdir confluent-hub-components

confluent-hub install --component-dir ./confluent-hub-components  --no-prompt confluentinc/kafka-connect-github:latest  

onfluent-hub install --component-dir ./confluent-hub-components --no-prompt confluentinc/kafka-connect-s3-source:1.3.2
```

Spin up docker compose stack 

```
docker-compose up -d
```

Update GH Token in the connector configuration and deploy the connector to start sourcing data

```
curl -X PUT -H "Content-type: application/json" -d @connectors/gh-connector.json http://localhost:8083/connectors/gh-connector-avro/config
```

Create KSqlDB stream 

```
CREATE STREAM stargazers with (KAFKA_TOPIC='github-avro-stargazers', VALUE_FORMAT='AVRO');
```

Try a push query

```
SELECT data FROM stargazers EMIT CHANGES LIMIT 5;
```

Create topic backed stream to join users who starred Jackdaw and Kafka repositories 

```
create stream stargazers_aggregate with (kafka_topic='stargazers_results', value_format='avro', partitions='1') as select sgz.data->id as id, sgz.data->login as login, sgz.data->type as type from stargazers as sgz inner join stargazers_jackdaw as sjack within 10 days on sgz.data->login=sjack.data->login  partition by sgz.data->id emit changes;
```

### Troubleshoot

* Not getting any data out of KSqlDB queries

```
SET 'auto.offset.reset' = 'earliest';
```