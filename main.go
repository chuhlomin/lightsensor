package main

import (
	"bufio"
	"bytes"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/matryer/try.v1"
	"gopkg.in/yaml.v3"
)

const (
	EventPing  = "ping"
	EventState = "state"
	StateNanLx = "nan lx"
)

// Message from lunarsensor
type Message struct {
	Event string `yml:"event"`
	Data  Data   `yml:"data"`
	Retry int    `yml:"retry"`
	ID    int    `yml:"id"`
}

type Data struct {
	ID    string `yml:"id"`
	State string `yml:"state"`
	Value float  `yml:"value"`
}

type float float64

// custom YAML unmarshal
func (v *float) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return errors.Wrap(err, "failed to unmarshal value")
	}

	if s == "NaN" {
		*v = 0
		return nil
	}

	i, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse float")
	}

	*v = float(i)
	return nil
}

func main() {
	var (
		url, addr  string
		maxRetries int
		delay      time.Duration
	)

	flag.StringVar(&addr, "addr", ":8080", "address to listen on")
	flag.StringVar(&url, "url", "http://lunarsensor.local/events", "Lunarsensor URL")
	flag.IntVar(&maxRetries, "max-retries", 5, "Max retries for connecting to Lunarsensor")
	flag.DurationVar(&delay, "delay", 1*time.Minute, "Delay between retries")

	flag.Parse()

	if err := run(addr, url, maxRetries, delay); err != nil {
		log.Fatal(err)
	}
}

func run(addr, url string, maxRetries int, delay time.Duration) error {
	recordMetrics(url, maxRetries, delay)

	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}

var (
	lightLevelGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "light_level",
		Help: "Light level in lux",
	})
)

func recordMetrics(url string, maxRetries int, delay time.Duration) {
	go func() {
		msgs := make(chan *Message)
		errs := make(chan error, 1)

		log.Println("Starting")

		reader := mustCreateStreamReader(url, maxRetries, delay)
		go readStream(reader, msgs, errs)

		for {
			select {
			case m := <-msgs:
				if m.Event == EventState && m.Data.State != StateNanLx {
					lightLevelGauge.Set(float64(m.Data.Value))
				}
			case err := <-errs:
				log.Printf("Error: %s\n", err)
				reader := mustCreateStreamReader(url, maxRetries, delay)
				go readStream(reader, msgs, errs)
			}
		}
	}()
}

func mustCreateStreamReader(url string, maxRetries int, delay time.Duration) *bufio.Reader {
	var reader *bufio.Reader
	err := try.Do(func(attempt int) (retry bool, err error) {
		retry = attempt < maxRetries
		log.Printf("Connecting to Lunarsensor %q (attempt %d/%d)", url, attempt, maxRetries)
		reader, err = createStreamReader(url)
		if err != nil && retry {
			log.Printf("Sleep %s before retry %d\n", delay, attempt)
			time.Sleep(delay * time.Duration(attempt))
		}
		return
	})
	if err != nil {
		log.Fatalln("error:", err)
	}

	return reader
}

func createStreamReader(url string) (*bufio.Reader, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stream")
	}

	return bufio.NewReader(resp.Body), nil
}

func readStream(reader *bufio.Reader, msgs chan<- *Message, errs chan<- error) {
	message := bytes.NewBuffer(nil)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			errs <- errors.Wrap(err, "failed to read line")
			return
		}

		msg, err := processLine(message, line)
		if err != nil {
			errs <- errors.Wrap(err, "failed to process line")
			return
		}
		if msg != nil {
			msgs <- msg
		}
	}
}

func processLine(message *bytes.Buffer, line []byte) (*Message, error) {
	message.Write(line)

	lineLen := len(bytes.TrimSpace(line))
	if lineLen == 0 {
		// message is complete
		m, err := processMessage(message)
		if err != nil {
			return nil, err
		}
		message.Reset()
		return m, nil
	}

	return nil, nil
}

func processMessage(message *bytes.Buffer) (*Message, error) {
	var m Message
	if err := yaml.Unmarshal(message.Bytes(), &m); err != nil {
		return nil, err
	}

	return &m, nil
}
