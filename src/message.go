package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

type MessageBuilder struct {
	data string
}

type JSONFormat struct {
	Command string          `json:"command"`
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"`
}

const (
	jsonPlaceholder = "JSON::"
	txtPlaceholder  = "TXT::"
)

type FormattedMessage string

func (m *MessageBuilder) formatToJSON() FormattedMessage {
	return (FormattedMessage)(strings.Join([]string{jsonPlaceholder, m.data}, ""))
}

func (m *MessageBuilder) formatToTXT() FormattedMessage {
	return (FormattedMessage)(strings.Join([]string{txtPlaceholder, m.data}, ""))
}

func (m FormattedMessage) base64Encrypt() []byte {
	return ([]byte)(base64.StdEncoding.EncodeToString([]byte(m)))
}

func (m *MessageBuilder) format() []byte {
	if m.isInTXTFormat() {
		return FormattedMessage(m.data).base64Encrypt()
	}

	var jsonOrPlainString string

	if m.isInJSONFormat() {
		jsonOrPlainString = strings.Replace(m.data, jsonPlaceholder, "", 1)
	} else {
		jsonOrPlainString = m.data
	}

	p := MessageBuilder{data: jsonOrPlainString}

	rawJs := &JSONFormat{}
	if json.Unmarshal([]byte(p.data), rawJs) == nil {
		return p.formatToJSON().base64Encrypt()
	}
	return p.formatToTXT().base64Encrypt()
}

func (m *MessageBuilder) isInTXTFormat() bool {
	return strings.Contains(m.data, txtPlaceholder)
}

func (m *MessageBuilder) isInJSONFormat() bool {
	return strings.Contains(m.data, jsonPlaceholder)
}

func (m *MessageBuilder) isPlainFormat() bool {
	return !(m.isInJSONFormat() || m.isInTXTFormat())
}

func (m *MessageBuilder) getCommand() (string, error) {
	if m.isInTXTFormat() {
		strippedMsg := strings.Replace(m.data, txtPlaceholder, "", 1)
		splittedMsg := strings.Split(strippedMsg, "::")

		return splittedMsg[1], nil

	} else if m.isInJSONFormat() {
		strippedMsg := strings.Replace(m.data, jsonPlaceholder, "", 1)

		parsedJson := &JSONFormat{}

		if json.Unmarshal([]byte(strippedMsg), parsedJson) == nil {
			return parsedJson.Command, nil
		}

		return "", errors.New("failed to parse the message")

	} else {
		return "", nil
	}
}

func (m *MessageBuilder) getVersion() (string, error) {
	if m.isInTXTFormat() {
		strippedMsg := strings.Replace(m.data, txtPlaceholder, "", 1)
		splittedMsg := strings.Split(strippedMsg, "::")

		return splittedMsg[0], nil

	} else if m.isInJSONFormat() {
		strippedMsg := strings.Replace(m.data, jsonPlaceholder, "", 1)

		parsedJson := &JSONFormat{}

		if json.Unmarshal([]byte(strippedMsg), parsedJson) == nil {
			return parsedJson.Version, nil
		}

		return "", errors.New("failed to parse the message")

	} else {
		return "", nil
	}
}
