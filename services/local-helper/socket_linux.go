//go:build linux

package helper

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"
	contracts "zyposoft.local/operations/packages/management-contracts"
	"zyposoft.local/operations/packages/unixpeer"
)

func (h *Helper) Serve(ctx context.Context, path string) error {
	return h.ServeWithLimit(ctx, path, 16)
}

// ServeWithLimit bounds accepted connections, including callers stalled on a
// partial frame. Cancellation closes active sockets before waiting for workers.
func (h *Helper) ServeWithLimit(ctx context.Context, path string, limit int) error {
	if limit < 1 || limit > 256 {
		return errors.New("CONNECTION_LIMIT")
	}
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	slots := make(chan struct{}, limit)
	l, err := unixpeer.Listen(path)
	if err != nil {
		return err
	}
	defer l.Close()
	stop := context.AfterFunc(ctx, func() { l.Close() })
	defer stop()
	for {
		c, err := l.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		select {
		case slots <- struct{}{}:
			workers.Add(1)
			go func() { defer workers.Done(); defer func() { <-slots }(); h.handleConn(ctx, c) }()
		default:
			c.Close()
		}
	}
}
func (h *Helper) handleConn(parent context.Context, c *net.UnixConn) {
	defer c.Close()
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { c.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if c.SetDeadline(deadline) != nil {
		return
	}
	if (unixpeer.Peer{UID: h.uid, GID: h.gid}).Check(c) != nil {
		_, _ = io.WriteString(c, "UNAUTHORISED_PEER\n")
		return
	}
	reader := bufio.NewReader(c)
	var size uint32
	if binary.Read(reader, binary.BigEndian, &size) != nil || size == 0 || size > contracts.MaxBytes {
		_, _ = io.WriteString(c, "INVALID_REQUEST\n")
		return
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return
	}
	if h.products != nil {
		_ = json.NewEncoder(c).Encode(h.dispatch(ctx, data))
		return
	}
	_, _ = io.WriteString(c, h.apply(data, h.uid, h.gid)+"\n")
}
