# Simple rate-limited network connections for Go

[![GoDoc](https://godoc.org/github.com/phf/go-ratelimit/ratelimit?status.png)](http://godoc.org/github.com/phf/go-ratelimit/ratelimit)

See http://godoc.org/github.com/phf/go-ratelimit/ratelimit for the documentation.

## Background

I am working on a simple network application that needs to ensure
that clients don't overwhelm it with traffic.
However, a complete traffic-shaper seemed overkill, so I wrote
this simple wrapper on top of the existing net.Conn abstractions.

Each connection can be given a rate limit in terms of bytes/second
and we'll approximate that by keeping track of time and data
transferred, sleeping when things are getting too fast for comfort.

Hardly rocket science, and probably not the best way of doing
things, but it seems to work fine.
