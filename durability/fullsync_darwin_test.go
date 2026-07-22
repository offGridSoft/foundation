//go:build darwin

package durability

import (
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
)

func TestDarwinFullSyncFallbackClassifierHostileErrnoTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "p01_einval", err: syscall.EINVAL, want: true},
		{name: "p02_enotty", err: syscall.ENOTTY, want: true},
		{name: "p03_enotsup", err: syscall.ENOTSUP, want: true},
		{name: "p04_eopnotsupp", err: syscall.EOPNOTSUPP, want: true},
		{name: "p05_enodev", err: syscall.ENODEV, want: true},
		{name: "p06_ebadf", err: syscall.EBADF, want: true},
		{name: "p07_wrapped_einval", err: &os.PathError{Op: "x", Path: "/x", Err: syscall.EINVAL}, want: true},
		{name: "p08_wrapped_enotty", err: &os.PathError{Op: "x", Path: "/x", Err: syscall.ENOTTY}, want: true},
		{name: "p09_double_wrapped_enotsup", err: &os.PathError{Op: "x", Path: "/x", Err: &os.PathError{Op: "y", Path: "/y", Err: syscall.ENOTSUP}}, want: true},
		{name: "p10_double_wrapped_ebadf", err: &os.PathError{Op: "x", Path: "/x", Err: &os.PathError{Op: "y", Path: "/y", Err: syscall.EBADF}}, want: true},
		{name: "n01_nil"},
		{name: "n02_eio", err: syscall.EIO},
		{name: "n03_eagain", err: syscall.EAGAIN},
		{name: "n04_enospc", err: syscall.ENOSPC},
		{name: "n05_eintr", err: syscall.EINTR},
		{name: "n06_eperm", err: syscall.EPERM},
		{name: "n07_eacces", err: syscall.EACCES},
		{name: "n08_enxio", err: syscall.ENXIO},
		{name: "n09_edquot", err: syscall.EDQUOT},
		{name: "n10_erofs", err: syscall.EROFS},
		{name: "b01_eof", err: io.EOF},
		{name: "b02_wrapped_eio", err: &os.PathError{Op: "x", Path: "/x", Err: syscall.EIO}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isFcntlUnsupported(tc.err)
			if got != tc.want {
				t.Fatalf("isFcntlUnsupported(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
	if !errors.Is(&os.PathError{Op: "x", Path: "/x", Err: syscall.ENOSPC}, syscall.ENOSPC) {
		t.Fatalf("errors.Is(wrapped ENOSPC, ENOSPC) = false, want true")
	}
}
