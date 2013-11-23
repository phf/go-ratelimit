// Copyright (c) 2013, Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file.

// Some inspiration for this test strategy comes from the
// src/pkg/net/conn_test.go example in the Go library.

package ratelimit

import (
	"io"
	"net"
	"testing"
	"time"
)

// The idea here is to create a connection with the given
// rate limits (rlim, wlim), send npack messages of length
// lpack from client to server, and finally have the server
// check whether it took within 20% of the expected duration.
//
// Testing against absolute times is a bad idea, especially
// if we want to test the case without rate limits as well.
// In fact I had to remove that case because it wouldn't
// run even remotely reliably. I am open to suggestions.

type testCase struct {
	net   string
	addr  string
	rlim  int
	wlim  int
	npack int
	lpack int
	total time.Duration
}

const accuracy = 0.2

var tests = []testCase{
//	{"tcp", "127.0.0.1:8080", 0, 0, 8, 1024, time.Duration(13 * time.Microsecond)},
	{"tcp", "127.0.0.1:8080", 4096, 0, 8, 1024, time.Duration(2 * time.Second)},
	{"tcp", "127.0.0.1:8080", 0, 4096, 8, 1024, time.Duration(2 * time.Second)},
	{"tcp", "127.0.0.1:8080", 2048, 4096, 8, 1024, time.Duration(4 * time.Second)},
	{"tcp", "127.0.0.1:8080", 4096, 2048, 8, 1024, time.Duration(4 * time.Second)},
	{"tcp", "127.0.0.1:8080", 16384, 0, 128, 16384, time.Duration(128 * time.Second)},
}

func TestBoundaries(t *testing.T) {
	_, err := New(nil, -1, 10)
	if err == nil {
		t.Errorf("expected New to fail but it didn't")
	}
	_, err = New(nil, 10, -1)
	if err == nil {
		t.Errorf("expected New to fail but it didn't")
	}
}

func TestConnections(t *testing.T) {
	for _, info := range tests {
		if testing.Short() && info.npack*info.lpack > 1024*1024 {
		        t.Skip("skipping test in short mode")
		        continue
		}
		testConnection(t, info)
	}
}

func testConnection(t *testing.T, info testCase) {
	// start the client (which will wait a bit before connecting)
	go testClient(t, info)
	// listen
	l, err := net.Listen(info.net, info.addr)
	if err != nil {
		t.Fatal(err)
	}
	// accept
	c, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	// wrap connection in rate limiter
	rlc, err := New(c, info.rlim, 0)
	if err != nil {
		t.Fatal(err)
	}
	// the actual experiment
	buf := make([]byte, info.npack*info.lpack)
	start := time.Now()
	n, err := io.ReadFull(rlc, buf)
	if err != nil {
		t.Error(err)
	}
	if n != len(buf) {
		t.Errorf("read %d bytes instead of %d bytes", n, len(buf))
	}
	duration := float64(time.Since(start).Nanoseconds())
	rlc.Close()
	l.Close()
	// check if we're "close" regarding timing
	expected := float64(info.total.Nanoseconds())
	lower := (1 - accuracy) * expected
	upper := (1 + accuracy) * expected
	if lower > duration || duration > upper {
		t.Errorf("expected around %f (%f..%f) but got %f", expected, lower, upper, duration)
	}
}

func testClient(t *testing.T, info testCase) {
	// make sure the server has time to come up
	time.Sleep(500 * time.Millisecond)
	// connect to the server
	c, err := net.Dial(info.net, info.addr)
	if err != nil {
		t.Fatal(err)
	}
	// wrap connection in rate limiter
	rlc, err := New(c, 0, info.wlim)
	if err != nil {
		t.Fatal(err)
	}
	// dump data into connection
	for i := 0; i < info.npack; i++ {
		data := make([]byte, info.lpack)
		n, err := rlc.Write(data)
		if err != nil {
			t.Error(err)
		}
		if n != info.lpack {
			t.Errorf("wrote %d bytes instead of %d bytes", n, info.lpack)
		}
	}
	// close the connection
	rlc.Close()
}
