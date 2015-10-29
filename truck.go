package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type payload struct {
	esIndex string
	esType  string
	data    map[string]interface{}
}

var (
	addr  = flag.String("listen", "0.0.0.0:5000", "hostname:port of listen address")
	es    = flag.String("elasticsearch", "127.0.0.1:9200", "hostname:port of elasticsearch")
	queue chan payload
)

func main() {
	flag.Parse()
	getEnv(addr, "LISTEN_ADDR")
	getEnv(es, "ELASTICSEARCH_ADDR")

	queue = make(chan payload, 10000)

	go listenUdp()
	process()
}

// Listens for messages on a udp socket and sends them ahead for processing.
func listenUdp() {
	laddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalf("unable to resolve listen address %s: %s\n", *addr, err)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatalf("unable to listen on %s: %s\n", *addr, err)
	}

	defer conn.Close()

	fmt.Printf("listening on udp %s\n", *addr)

	msg := make([]byte, 32768)
	for {
		n, raddr, err := conn.ReadFrom(msg)
		if err != nil {
			if err != io.EOF {
				log.Printf("warn: unable to read from %s: %s\n", *addr, err)
			}
			continue
		}

		decode(parseHost(raddr.String()), msg[0:n])
	}
}

// Works the payload queue, inserting messages into elasticsearch.
// Volume is bound by latency and may need concurrent workers for heavy workloads.
func process() {
	var buf []byte
	var err error

	for payload := range queue {
		// Marshal the payload data to JSON
		buf, err = json.Marshal(payload.data)
		if err != nil {
			log.Printf("invalid payload: %s\n", err)
			continue
		}

		// Establish the Elasticsearch PUT url
		url := fmt.Sprintf("http://%s/%s/%s", *es, payload.esIndex, payload.esType)

		// Create a new HTTP request
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
		if err != nil {
			continue
		}

		// Set the Content-Type
		req.Header.Set("Content-Type", "application/json")

		// Perform the HTTP request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("unable to write payload to elasticsearch: %s", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			rdata := make(map[string]interface{})
			if err := json.NewDecoder(resp.Body).Decode(&rdata); err != nil {
				log.Printf("unable to decode response body: %s", err)
			} else {
				log.Printf("unexpected status code %s for response: %s", resp.StatusCode, rdata)
			}
		}
	}
}

// Transforms a given message into a payload object, adding metadata and
// queuing it up for delivery.
func decode(host string, buf []byte) {
	// This is the payload we'll store with elasticsearch.
	payload := payload{
		esIndex: currentIndex(),
		esType:  "logs",
		data:    make(map[string]interface{}),
	}

	// Try to detect if the buffer is JSON. Decode into the payload if so.
	data := make(map[string]interface{})
	if err := json.Unmarshal(buf, &data); err == nil {
		payload.data = data
	} else {
		payload.data["message"] = string(buf)
	}

	// Add other required fields
	payload.data["host"] = host
	payload.data["@timestamp"] = time.Now().Format(time.RFC3339)
	payload.data["@version"] = 1

	queue <- payload
}

func currentIndex() string {
	return fmt.Sprintf("logstash-%s", time.Now().Format("2006.01.02"))
}

// Updates the string in p to a given environment variable if set.
func getEnv(p *string, key string) {
	if s := os.Getenv(key); s != "" {
		*p = s
	}
}

// Take just the hostname of a host:pair address. Not IPv6 friendly.
func parseHost(s string) string {
	return strings.Split(s, ":")[0]
}
