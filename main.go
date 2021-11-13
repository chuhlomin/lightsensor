package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
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

	i, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return errors.Wrap(err, "failed to parse float")
	}

	*v = float(i)
	return nil
}

func main() {
	log.Println("Starting")

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		url        = "http://192.168.0.221/events"
		maxRetries = 5
		msgs       = make(chan *Message)
		errs       = make(chan error, 1)
	)

	reader := mustCreateStreamReader(url, maxRetries)
	go readStream(reader, msgs, errs)

	for {
		select {
		case m := <-msgs:
			if m.Event == EventState && m.Data.State != StateNanLx {
				fmt.Printf("%d %f\n", time.Now().UnixMilli(), m.Data.Value)
			}
		case err := <-errs:
			log.Printf("Error: %s\n", err)
			reader := mustCreateStreamReader(url, maxRetries)
			go readStream(reader, msgs, errs)
		}
	}
}

func mustCreateStreamReader(url string, maxRetries int) *bufio.Reader {
	var reader *bufio.Reader
	err := try.Do(func(attempt int) (retry bool, err error) {
		retry = attempt < maxRetries
		reader, err = createStreamReader(url)
		if err != nil && retry {
			time.Sleep(1 * time.Minute)
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
