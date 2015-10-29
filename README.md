# Truck

Truck ships your logs from a udp socket to an Elasticsearch cluster. It's a extremely simple replacement for Logstash with no post-processing capabilities beyond adding basic metadata.

# Running it with Docker

`docker run -p 5000:5000 -e ELASTICSEARCH_ADDR=127.0.0.1:9200 jcoene/truck`

# License

MIT

