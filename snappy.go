package nsq

import (
	"bufio"
	"net"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
)

func upgradeSnappy(c *Conn) (*Conn, error) {
	netConn := net.Conn(c)
	conn := &Conn{
		conn: c,
		rbuf: bufio.NewReaderSize(snappy.NewReader(netConn), connBufSize),
		wbuf: bufio.NewWriterSize(snappy.NewBufferedWriter(netConn), connBufSize),
	}
	frame, err := conn.ReadFrame()
	if err != nil {
		return c, err
	}
	if frame.FrameType() != FrameTypeResponse {
		if frame.(Response) != OK {
			return c, errors.New("invalid response from Snappy upgrade")
		}
	}
	return conn, err
}
