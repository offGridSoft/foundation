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

func TestRetryInterruptedFcntlExhaustsInterruptsAndPreservesTerminalErrno(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		sequence  []syscall.Errno
		wantCalls int
		wantErr   error
	}{
		{name: "immediate success", sequence: []syscall.Errno{0}, wantCalls: 1},
		{name: "one interrupt then success", sequence: []syscall.Errno{syscall.EINTR, 0}, wantCalls: 2},
		{name: "three interrupts then success", sequence: []syscall.Errno{syscall.EINTR, syscall.EINTR, syscall.EINTR, 0}, wantCalls: 4},
		{name: "immediate input output failure", sequence: []syscall.Errno{syscall.EIO}, wantCalls: 1, wantErr: syscall.EIO},
		{name: "interrupt then input output failure", sequence: []syscall.Errno{syscall.EINTR, syscall.EIO}, wantCalls: 2, wantErr: syscall.EIO},
		{name: "interrupt then no space failure", sequence: []syscall.Errno{syscall.EINTR, syscall.ENOSPC}, wantCalls: 2, wantErr: syscall.ENOSPC},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			err := retryInterruptedFcntl(func() syscall.Errno {
				errno := testCase.sequence[calls]
				calls++
				return errno
			})
			if calls != testCase.wantCalls {
				t.Fatalf("retryInterruptedFcntl() calls = %d, want %d", calls, testCase.wantCalls)
			}
			if testCase.wantErr == nil && err != nil {
				t.Fatalf("retryInterruptedFcntl() error = %v, want nil", err)
			}
			if testCase.wantErr != nil && !errors.Is(err, testCase.wantErr) {
				t.Fatalf("retryInterruptedFcntl() error = %v, want errors.Is %v", err, testCase.wantErr)
			}
		})
	}
}
