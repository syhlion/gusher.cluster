package requestwork

import (
	"errors"
	"net"
	"time"
)

type TimeoutConn struct {
	net.Conn
	readTimeout, writeTimeout time.Duration
}

var invalidOperationError = errors.New("TimeoutConn does not support or allow .SetDeadline operations")

func NewTimeoutConn(conn net.Conn, ioTimeout time.Duration) (*TimeoutConn, error) {
	return NewTimeoutConnReadWriteTO(conn, ioTimeout, ioTimeout)
}

func NewTimeoutConnReadWriteTO(conn net.Conn, readTimeout, writeTimeout time.Duration) (*TimeoutConn, error) {
	this := &TimeoutConn{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
	now := time.Now()
	err := this.Conn.SetReadDeadline(now.Add(this.readTimeout))
	if err != nil {
		return nil, err
	}
	err = this.Conn.SetWriteDeadline(now.Add(this.writeTimeout))
	if err != nil {
		return nil, err
	}
	return this, nil
}

func NewTimeoutConnDial(network, addr string, ioTimeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout(network, addr, ioTimeout)
	if err != nil {
		return nil, err
	}
	if conn, err = NewTimeoutConn(conn, ioTimeout); err != nil {
		return nil, err
	}
	return conn, nil
}

func (t *TimeoutConn) Read(data []byte) (int, error) {
	t.Conn.SetReadDeadline(time.Now().Add(t.readTimeout))
	return t.Conn.Read(data)
}

func (t *TimeoutConn) Write(data []byte) (int, error) {
	t.Conn.SetWriteDeadline(time.Now().Add(t.writeTimeout))
	return t.Conn.Write(data)
}

func (t *TimeoutConn) SetDeadline(time time.Time) error {
	return invalidOperationError
}

func (t *TimeoutConn) SetReadDeadline(time time.Time) error {
	return invalidOperationError
}

func (t *TimeoutConn) SetWriteDeadline(time time.Time) error {
	return invalidOperationError
}
