//go:build linux

package unixpeer

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Peer struct{ UID, GID uint32 }

func (p Peer) Check(c *net.UnixConn) error {
	fd, err := c.SyscallConn()
	if err != nil {
		return err
	}
	var u *unix.Ucred
	var sockErr error
	err = fd.Control(func(fd uintptr) { u, sockErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
	if err != nil || sockErr != nil || u == nil || p.UID != u.Uid || p.GID != u.Gid {
		return errors.New("UNAUTHORISED_PEER")
	}
	return nil
}

type Endpoint struct {
	Path string
	Peer Peer
}

func (e Endpoint) Dial(ctx context.Context) (net.Conn, error) {
	c, err := new(net.Dialer).DialContext(ctx, "unix", e.Path)
	if err != nil {
		return nil, err
	}
	if err = e.Peer.Check(c.(*net.UnixConn)); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

type OwnedListener struct {
	*net.UnixListener
	lock     *os.File
	once     sync.Once
	closeErr error
}

func (l *OwnedListener) Close() error {
	l.once.Do(func() { l.closeErr = l.UnixListener.Close(); l.lock.Close() })
	return l.closeErr
}
func Listen(path string) (*OwnedListener, error) {
	dir, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	owner, ok := dir.Sys().(*syscall.Stat_t)
	if !ok || !dir.IsDir() || dir.Mode().Perm()&0022 != 0 || owner.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("UNSAFE_SOCKET_DIRECTORY")
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	info, err := lock.Stat()
	if err != nil {
		lock.Close()
		return nil, err
	}
	lockOwner, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || lockOwner.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0022 != 0 {
		lock.Close()
		return nil, errors.New("UNSAFE_SOCKET_LOCK")
	}
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			lock.Close()
		}
	}()
	if info, e := os.Lstat(path); e == nil {
		socketOwner, ok := info.Sys().(*syscall.Stat_t)
		if info.Mode()&os.ModeSocket == 0 || !ok || socketOwner.Uid != uint32(os.Geteuid()) {
			return nil, errors.New("SOCKET_PATH_OCCUPIED")
		}
		c, e := net.DialTimeout("unix", path, 100*time.Millisecond)
		if e == nil {
			c.Close()
			return nil, errors.New("SOCKET_ACTIVE")
		}
		if !errors.Is(e, syscall.ECONNREFUSED) {
			return nil, e
		}
		if err = os.Remove(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(path, 0660); err != nil {
		l.Close()
		return nil, err
	}
	success = true
	return &OwnedListener{UnixListener: l, lock: lock}, nil
}

type Listener struct {
	*OwnedListener
	Allowed Peer
}

func (l Listener) Accept() (net.Conn, error) {
	for {
		c, err := l.AcceptUnix()
		if err != nil {
			return nil, err
		}
		if l.Allowed.Check(c) == nil {
			return c, nil
		}
		c.Close()
	}
}
