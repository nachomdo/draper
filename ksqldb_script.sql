SET 'auto.offset.reset' = 'earliest';

CREATE STREAM stargazers_kafka with (KAFKA_TOPIC='github-avro-stargazers-kafka', VALUE_FORMAT='AVRO');

CREATE STREAM stargazers_jackdaw with (KAFKA_TOPIC='github-avro-stargazers-jackdaw', VALUE_FORMAT='AVRO');

CREATE STREAM stargazers_aggregate WITH (kafka_topic='stargazers-results', value_format='json', partitions='1')
AS SELECT sgz.data->id AS id, sgz.data->login AS login, sgz.data->type AS type
FROM stargazers_kafka AS sgz
INNER JOIN stargazers_jackdaw AS sjack WITHIN 10 DAYS ON sgz.data->login=sjack.data->login
PARTITION BY sgz.data->id EMIT CHANGES;

