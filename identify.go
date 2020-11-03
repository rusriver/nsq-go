package nsq

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/segmentio/encoding/json"
)

// Identify represents the IDENTIFY command.
type Identify struct {
	// ClientID should be set to a unique identifier representing the client.
	ClientID string

	// Hostname represents the hostname of the client, by default it is set to
	// the value returned by os.Hostname is used.
	Hostname string

	// UserAgent represents the type of the client, by default it is set to
	// nsq.DefaultUserAgent.
	UserAgent string

	// MessageTimeout can bet set to configure the server-side message timeout
	// for messages delivered to this consumer.  By default it is not sent to
	// the server.
	MessageTimeout time.Duration
}

type IdentityResponse struct {
	MaxRdyCount  int  `json:"max_rdy_count"`
	AuthRequired bool `json:"auth_required"`
}

type identifyBody struct {
	ClientID       string `json:"client_id,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	UserAgent      string `json:"user_agent,omitempty"`
	MessageTimeout int    `json:"msg_timeout,omitempty"`
	Negotiation    bool   `json:"feature_negotiation,omitempty"`
}

const CommandIdentify = "IDENTIFY"

// Name returns the name of the command in order to satisfy the Command
// interface.
func (c Identify) Name() string {
	return CommandIdentify
}

// Write serializes the command to the given buffered output, satisfies the
// Command interface.
func (c Identify) Write(w *bufio.Writer) (err error) {
	var data []byte
	body := identifyBody{
		ClientID:       c.ClientID,
		Hostname:       c.Hostname,
		UserAgent:      c.UserAgent,
		MessageTimeout: int(c.MessageTimeout / time.Millisecond),
		Negotiation:    true,
	}

	if data, err = json.Marshal(body); err != nil {
		return
	}

	if _, err = w.WriteString("IDENTIFY\n"); err != nil {
		err = errors.Wrap(err, "writing IDENTIFY command")
		return
	}

	if err = binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		err = errors.Wrap(err, "writing IDENTIFY body size")
		return
	}

	if _, err = w.Write(data); err != nil {
		err = errors.Wrap(err, "writing IDENTIFY body data")
		return
	}

	return
}

func readIdentify(r *bufio.Reader) (cmd Identify, err error) {
	var body identifyBody

	if body, err = readIdentifyBody(r); err != nil {
		return
	}

	cmd = Identify{
		ClientID:       body.ClientID,
		Hostname:       body.Hostname,
		UserAgent:      body.UserAgent,
		MessageTimeout: time.Millisecond * time.Duration(body.MessageTimeout),
	}
	return
}

func readIdentityResponse(conn *Conn) (IdentityResponse, error) {
	var ir IdentityResponse

	frame, err := conn.ReadFrame()
	if err != nil {
		return ir, err
	}
	resp, ok := frame.(Response)
	if !ok {
		return ir, errors.New("invalid identify response")
	}

	switch resp {
	case OK:
		return ir, nil
	default:
		if err := json.Unmarshal([]byte(resp), &ir); err != nil {
			return ir, err
		}
	}
	return ir, nil
}

func readIdentifyBody(r *bufio.Reader) (body identifyBody, err error) {
	var size uint32
	var data []byte

	if err = binary.Read(r, binary.BigEndian, &size); err != nil {
		err = errors.Wrap(err, "reading IDENTIFY body size")
		return
	}

	data = make([]byte, int(size))

	if _, err = io.ReadFull(r, data); err != nil {
		err = errors.Wrap(err, "reading IDENTIFY body data")
		return
	}

	if err = json.Unmarshal(data, &body); err != nil {
		err = errors.Wrap(err, "decoding IDENTIFY body")
		return
	}

	return
}

func setIdentifyDefaults(c Identify) Identify {
	if len(c.UserAgent) == 0 {
		c.UserAgent = DefaultUserAgent
	}

	if len(c.Hostname) == 0 {
		c.Hostname, _ = os.Hostname()
	}

	return c
}
