package nsq

import (
	"bufio"
	"compress/flate"
	"net"

	"github.com/pkg/errors"
)




func upgradeDeflate(c *Conn, level int) (*Conn, error) {
	netConn := net.Conn(c)
	fw, _ := flate.NewWriter(netConn, level)
	conn := &Conn{
		conn: c,
		rbuf: bufio.NewReaderSize(flate.NewReader(netConn), connBufSize),
		wbuf: bufio.NewWriterSize(fw, connBufSize),
	}
	frame, err := conn.ReadFrame()
	if err != nil {
		return c, err
	}
	if frame.FrameType() != FrameTypeResponse {
		if frame.(Response) != OK {
			return c, errors.New("invalid response from Deflate upgrade")
		}
	}
	return conn, nil
}
