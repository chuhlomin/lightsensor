package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestFloat(t *testing.T) {
	var v float
	err := yaml.Unmarshal([]byte("0.839524"), &v)
	assert.Nil(t, err)
	assert.Equal(t, float(0.83952397108078), v)

	err = yaml.Unmarshal([]byte("NaN"), &v)
	assert.Nil(t, err)
	assert.Equal(t, float(0), v)

	err = yaml.Unmarshal([]byte("false"), &v)
	assert.NotNil(t, err)
}

func TestProcessMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected Message
	}{
		{
			input:    "",
			expected: Message{},
		},
		{
			input: `event: state
data: {"id":"sensor-ambient_light_tsl2591","state":"1","value":0.839524}`,
			expected: Message{
				Event: "state",
				Data: Data{
					ID:    "sensor-ambient_light_tsl2591",
					State: "1",
					Value: 0.83952397108078,
				},
			},
		},
		{
			input: `retry: 30000
id: 5098007
event: ping
data:`,
			expected: Message{
				Event: "ping",
				ID:    5098007,
				Retry: 30000,
				Data:  Data{},
			},
		},
		{
			input: `event: state
data: {"id":"sensor-ambient_light_tsl2561","state":"nan lx","value":NaN}`,
			expected: Message{
				Event: "state",
				Data: Data{
					ID:    "sensor-ambient_light_tsl2561",
					State: "nan lx",
					Value: 0.0,
				},
			},
		},
	}

	message := bytes.NewBuffer(nil)

	for _, test := range tests {
		message.Write([]byte(test.input))

		m, err := processMessage(message)
		assert.Nil(t, err)

		assert.EqualValues(t, test.expected, *m)

		message.Reset()
	}
}

func TestProcessMessageUnmarshalError(t *testing.T) {
	message := bytes.NewBuffer(nil)
	message.Write([]byte("invalid yaml"))

	_, err := processMessage(message)
	assert.NotNil(t, err)
}

func TestReadStream(t *testing.T) {
	tests := []struct {
		input                    string
		expectedMsgs             []Message
		expectedErrMessagePrefix string
	}{
		{
			input: `event: state
data: {"id":"sensor-ambient_light_tsl2591","state":"1","value":0.839524}

event: state
data: {"id":"sensor-ambient_light_tsl2561","state":"nan lx","value":NaN}

`,
			expectedMsgs: []Message{
				{
					Event: "state",
					Data: Data{
						ID:    "sensor-ambient_light_tsl2591",
						State: "1",
						Value: 0.83952397108078,
					},
				},
				{
					Event: "state",
					Data: Data{
						ID:    "sensor-ambient_light_tsl2561",
						State: "nan lx",
						Value: 0.0,
					},
				},
			},
			expectedErrMessagePrefix: "failed to read line: EOF",
		},
		{
			input: `invalid yaml

`,
			expectedMsgs:             []Message{},
			expectedErrMessagePrefix: "failed to process line: yaml: unmarshal errors",
		},
	}

	msgs := make(chan *Message)
	errs := make(chan error)

	for _, test := range tests {
		reader := bufio.NewReader(bytes.NewReader([]byte(test.input)))

		go readStream(reader, msgs, errs)

		messages := []Message{}

	L:
		for {
			fmt.Print(".")
			select {
			case msg := <-msgs:
				log.Printf("%+v\n", msg)
				messages = append(messages, *msg)
			case err := <-errs:
				log.Printf("%+v\n", err)
				assert.True(t, strings.HasPrefix(err.Error(), test.expectedErrMessagePrefix), err.Error())

				break L
			}
		}

		assert.Equal(t, test.expectedMsgs, messages)
	}
}
