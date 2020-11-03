package nsq

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	"io/ioutil"
	"net"
	"time"
)

type TLSConfig struct {
	RootFile string
	CertFile string
	KeyFile  string
}

func NewTLSConfig(config TLSConfig) (*tls.Config, error) {
	var tlsConf *tls.Config
	if len(config.CertFile) > 0 && len(config.KeyFile) > 0 {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConf = &tls.Config{
			MinVersion:   tls.VersionTLS10,
			Certificates: []tls.Certificate{cert},
		}
		if len(config.RootFile) > 0 {
			tlsCertPool := x509.NewCertPool()
			caCertFile, err := ioutil.ReadFile(config.RootFile)
			if err != nil {
				return nil, err
			}
			if !tlsCertPool.AppendCertsFromPEM(caCertFile) {
				return nil, errors.New("failed to append certificates from Certificate Authority file")
			}
			tlsConf.RootCAs = tlsCertPool
		}
	}
	return tlsConf, nil
}

func upgradeTLS(c *Conn, addr string, config *tls.Config) (*Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return c, nil
	}
	conf := &tls.Config{}
	if config != nil {
		conf = config.Clone()
	}
	conf.ServerName = host

	tlsConn := tls.Client(c.conn, conf)
	if err = tlsConn.SetDeadline(time.Time{}); err != nil {
		tlsConn.Close()
		return c, errors.Wrap(err, "resetting deadlines on TLS connection to tcp://"+addr)
	}
	if err := tlsConn.Handshake(); err != nil {
		return c, errors.WithMessage(err, "trying to handshake")
	}
	conn := &Conn{
		conn: tlsConn,
		rbuf: bufio.NewReaderSize(tlsConn, connBufSize),
		wbuf: bufio.NewWriterSize(tlsConn, connBufSize),
	}

	frame, err := conn.ReadFrame()
	if err != nil {
		return c, err
	}
	if frame.FrameType() != FrameTypeResponse {
		if frame.(Response) != OK {
			return c, errors.New("invalid response from TLS upgrade")
		}
	}
	return conn, nil
}
