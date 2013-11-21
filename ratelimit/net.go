// Copyright (c) 2013, Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file.

// Package ratelimit provides a rate-limiting wrapper for net.Conn
// network connections.
//
// Our approach to rate-limiting is somewhat crude: We simply sleep
// for an appropriate amount of time after each Read or Write. This
// assumes (for Read limiting anyway) that TCP's congestion control
// mechanisms eventually "catch on" and reduce the amount of traffic
// they send our way.
package ratelimit

import (
	"errors"
	"net"
	"time"
)

type RateLimitedConn struct {
	net.Conn               // underlying network connection
	rlim, wlim   int       // in bytes/second, 0 means no limit
	rtime, wtime time.Time // time of last actual read/write
}

// New returns a rate-limited connection based on the given connection.
// The limits are specified in bytes per second (bps) and 0 means no
// limit.
//
// Note that rate-limiting doesn't take connection deadlines into account
// (see SetDeadline, SetReadDeadline, and SetWriteDeadline) so be careful
// when using both.
func New(conn net.Conn, readLimit, writeLimit int) (rlc net.Conn, err error) {
	if readLimit < 0 || writeLimit < 0 {
		err = errors.New("read/write limits cannot be negative")
		return
	}
	rlc = RateLimitedConn{Conn: conn, rlim: readLimit, wlim: writeLimit}
	return
}

// Read reads data from the connection.
// If necessary this function will sleep for an appropriate amount
// of time to achieve the requested rate-limit.
func (rlc RateLimitedConn) Read(b []byte) (n int, err error) {
	// fast path if there is no limit
	if rlc.rlim <= 0 {
		n, err = rlc.Conn.Read(b)
		return
	}

	// lazy initialization
	if rlc.rtime.IsZero() {
		rlc.rtime = time.Now()
	}

	// perform the read operation
	n, err = rlc.Conn.Read(b)

	// how long since the last read?
	t := time.Now()
	d := t.Sub(rlc.rtime).Nanoseconds()

	// allowed time
	timePerByte := time.Second.Nanoseconds() / int64(rlc.rlim)
	timeForNBytes := timePerByte * int64(n)

	// sleep if we have to
	if n > 0 && d < timeForNBytes {
		time.Sleep(time.Duration(timeForNBytes - d))
	}

	// remember when last read finished
	rlc.rtime = t
	return
}

// Write writes data to the connection.
// If necessary this function will sleep for an appropriate amount
// of time to achieve the requested rate-limit.
func (rlc RateLimitedConn) Write(b []byte) (n int, err error) {
	// fast path if there is no limit
	if rlc.wlim <= 0 {
		n, err = rlc.Conn.Write(b)
		return
	}

	// lazy initialization
	if rlc.wtime.IsZero() {
		rlc.wtime = time.Now()
	}

	// perform the write operation
	n, err = rlc.Conn.Write(b)

	// how long since the last write?
	t := time.Now()
	d := t.Sub(rlc.wtime).Nanoseconds()

	// allowed time
	timePerByte := time.Second.Nanoseconds() / int64(rlc.wlim)
	timeForNBytes := timePerByte * int64(n)

	// sleep if we have to
	if n > 0 && d < timeForNBytes {
		time.Sleep(time.Duration(timeForNBytes - d))
	}

	// remember when last write finished
	rlc.wtime = t
	return
}

// SetReadLimit establishes a new limit (in bytes per second, 0 for
// no limit) for reading from this connection.
func (rlc RateLimitedConn) SetReadLimit(lim int) (err error) {
	if lim < 0 {
		err = errors.New("read limit cannot be negative")
		return
	}
	rlc.rlim = lim
	return
}

// SetWriteLimit establishes a new limit (in bytes per second, 0 for
// no limit) for writing to this connection.
func (rlc RateLimitedConn) SetWriteLimit(lim int) (err error) {
	if lim < 0 {
		err = errors.New("write limit cannot be negative")
		return
	}
	rlc.wlim = lim
	return
}
