package sys

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"unsafe"

	"github.com/coreos/go-systemd/v22/dbus"
	"golang.org/x/sys/unix"
)

// in some container environments we may be able to talk to the "host" systemd
// user instance, but we don't want to _use_ it in that case because it won't
// see our paths/etc.
var (
	ErrWrongNamespace      = fmt.Errorf("systemd instance is in a different namespace")
	errReflectForNamespace = fmt.Errorf("unable to extract dbus connection info: %w", ErrWrongNamespace)
)

// SystemdUserConn wraps [dbus.NewUserConnectionContext] but verifies that the
// connection is to a systemd instance in the same namespace.
//
// If the systemd instance is in a different namespace, or it runs into errors
// attemping the verification, then it will return an error wrapping
// [ErrWrongNamespace].
func SystemdUserConn(ctx context.Context) (_ *dbus.Conn, finalErr error) {
	conn, err := dbus.NewUserConnectionContext(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if finalErr != nil {
			conn.Close() //nolint:errcheck
		}
	}()

	// systemctl checks this by issuing `getsockopt(3, SOL_SOCKET, SO_PEERCRED,
	// ...)` and observing if it gets back a valid PID. The problem is that we
	// need to get the unix socket fd out of the dbus connection object, which is
	// obnoxious.
	v := reflect.ValueOf(conn).Elem()
	sc := deref(v.FieldByName("sysconn"))
	if !sc.IsValid() {
		return nil, errReflectForNamespace
	}
	ut := deref(sc.FieldByName("transport"))
	if !ut.IsValid() {
		return nil, errReflectForNamespace
	}
	uc := deref(ut.FieldByName("UnixConn"))
	if !uc.IsValid() {
		return nil, errReflectForNamespace
	}
	if uc.Type() != reflect.TypeFor[net.UnixConn]() {
		return nil, errReflectForNamespace
	}
	// dirty hack to get around unexported field restriction
	ucv, ok := launder(uc).Addr().Interface().(*net.UnixConn)
	if !ok {
		return nil, errReflectForNamespace
	}
	rawConn, err := ucv.SyscallConn()
	if err != nil {
		return nil, errReflectForNamespace
	}
	var credsErr error
	credsOK := false
	if err := rawConn.Control(func(fd uintptr) {
		if creds, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED); err != nil {
			credsErr = err
		} else if creds.Pid > 0 {
			credsOK = true
		}
	}); err != nil {
		return nil, fmt.Errorf("%w (%w)", errReflectForNamespace, err)
	}
	if credsErr != nil {
		return nil, fmt.Errorf("%w (%w)", errReflectForNamespace, credsErr)
	} else if !credsOK {
		return nil, errReflectForNamespace
	}

	return conn, nil
}

func deref(v reflect.Value) reflect.Value {
	for k := v.Kind(); k == reflect.Pointer || k == reflect.Interface; k = v.Kind() {
		v = v.Elem()
	}
	return v
}

func launder(v reflect.Value) reflect.Value {
	return deref(reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())))
}
