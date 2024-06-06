package wintun_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net/netip"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/lysShub/wintun-go"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/checksum"
	"gvisor.dev/gvisor/pkg/tcpip/header"
)

func randPort() int {
	for {
		port := uint16(rand.Uint32())
		if port > 2048 && port < 0xffff-0xff {
			return int(port)
		}
	}
}

var dllPath = `.\embed\wintun_amd64.dll`

func init() {
	switch runtime.GOARCH {
	case "amd64":
	case "386":
		dllPath = `.\embed\wintun_386.dll`
	case "arm":
		dllPath = `.\embed\wintun_arm.dll`
	case "arm64":
		dllPath = `.\embed\wintun_arm64.dll`
	default:
		panic("")
	}
}

func buildICMP(t require.TestingT, src, dst []byte, typ header.ICMPv4Type, msg []byte) []byte {
	require.Zero(t, len(msg)%4)

	var p = make([]byte, 28+len(msg))
	iphdr := header.IPv4(p)
	iphdr.Encode(&header.IPv4Fields{
		TOS:            0,
		TotalLength:    uint16(len(p)),
		ID:             uint16(rand.Uint32()),
		Flags:          0,
		FragmentOffset: 0,
		TTL:            128,
		Protocol:       uint8(header.ICMPv4ProtocolNumber),
		Checksum:       0,
		SrcAddr:        tcpip.AddrFromSlice(src),
		DstAddr:        tcpip.AddrFromSlice(dst),
	})
	iphdr.SetChecksum(^checksum.Checksum(p[:iphdr.HeaderLength()], 0))
	require.True(t, iphdr.IsChecksumValid())

	icmphdr := header.ICMPv4(iphdr.Payload())
	icmphdr.SetType(typ)
	icmphdr.SetIdent(0)
	icmphdr.SetSequence(0)
	icmphdr.SetChecksum(0)
	copy(icmphdr.Payload(), msg)
	icmphdr.SetChecksum(^checksum.Checksum(icmphdr, 0))
	return p
}

func Test_Example(t *testing.T) {
	// https://github.com/WireGuard/wintun/blob/master/example/example.c
	require.NoError(t, wintun.Load(wintun.DLL))

	// 10.6.7.7/24
	var addr = netip.PrefixFrom(
		netip.MustParseAddr("10.6.7.7"),
		24,
	)

	ap, err := wintun.CreateAdapter("testexample")
	require.NoError(t, err)
	defer ap.Close()

	luid, err := ap.GetAdapterLuid()
	require.NoError(t, err)
	err = luid.AddIPAddress(addr)
	require.NoError(t, err)

	// Send: ping -S 10.6.7.8 10.6.7.7
	var ch = make(chan struct{})
	defer func() { <-ch }()
	go func() {
		defer close(ch)

		pack := buildICMP(t,
			addr.Addr().Next().AsSlice(),
			addr.Addr().AsSlice(),
			header.ICMPv4Echo, []byte("1234"),
		)
		for {
			p, err := ap.Alloc(len(pack))
			if errors.Is(err, wintun.ErrAdapterClosed{}) {
				return
			}
			require.NoError(t, err)

			copy(p, pack)

			err = ap.Send(p)
			require.NoError(t, err)
			if errors.Is(err, os.ErrClosed) {
				return
			}
			time.Sleep(time.Second)
		}
	}()

	// Recv outgoing ICMP Echo-Reply packet
	for ok := false; !ok; {
		p, err := ap.Recv(context.Background())
		require.NoError(t, err)

		switch header.IPVersion(p) {
		case 4:
			iphdr := header.IPv4(p)

			ok = iphdr.SourceAddress().String() == "10.6.7.7" &&
				iphdr.DestinationAddress().String() == "10.6.7.8"
			if iphdr.TransportProtocol() == header.ICMPv4ProtocolNumber {
				icmp := header.ICMPv4(iphdr.Payload())

				ok = ok &&
					icmp.Type() == header.ICMPv4EchoReply &&
					string(icmp.Payload()) == "1234"
			} else {
				ok = false
			}
		default:
		}
		err = ap.Release(p)
		require.NoError(t, err)
	}
	require.NoError(t, ap.Close())
}

func Test_DriverVersion(t *testing.T) {
	t.Skip("can't get driver version")
	t.Run("mem", func(t *testing.T) {

		require.NoError(t, wintun.Load(wintun.DLL))

		ver, err := wintun.DriverVersion()
		require.NoError(t, err)
		t.Log(ver)
	})
	t.Run("file", func(t *testing.T) {
		require.NoError(t, wintun.Load(dllPath))

		ver, err := wintun.DriverVersion()
		require.NoError(t, err)
		t.Log(ver)
	})
}

func Test_Logger(t *testing.T) {
	t.Run("mem", func(t *testing.T) {
		require.NoError(t, wintun.Load(wintun.DLL))

		buff := bytes.NewBuffer(nil)
		log := slog.New(slog.NewJSONHandler(buff, nil))
		callback := wintun.DefaultCallback(log)

		err := wintun.SetLogger(callback)
		require.NoError(t, err)

		{
			w, err := wintun.CreateAdapter("testlogger")
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
		}

		require.Contains(t, buff.String(), "Creating")
	})
	t.Run("file", func(t *testing.T) {
		require.NoError(t, wintun.Load(dllPath))

		buff := bytes.NewBuffer(nil)
		log := slog.New(slog.NewJSONHandler(buff, nil))
		callback := wintun.DefaultCallback(log)

		err := wintun.SetLogger(callback)
		require.NoError(t, err)

		{
			w, err := wintun.CreateAdapter("testlogger")
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
		}

		require.Contains(t, buff.String(), "Creating")
	})
}

func Test_Load(t *testing.T) {
	var _ = windows.ERROR_RING2SEG_MUST_BE_MOVABLE

	t.Run("mem:load-fail", func(t *testing.T) {
		err := wintun.Load(make(wintun.Mem, 64))
		require.Error(t, err)
	})
	t.Run("file:load-fail", func(t *testing.T) {
		err := wintun.Load("./wintun.go")
		require.Error(t, err)
	})

	t.Run("load-fail/load", func(t *testing.T) {
		err := wintun.Load(make(wintun.Mem, 64))
		require.Error(t, err)

		require.NoError(t, wintun.Load(dllPath))

	})

	t.Run("load/load", func(t *testing.T) {
		require.NoError(t, wintun.Load(wintun.DLL))

		err := wintun.Load(wintun.DLL)
		require.True(t, errors.Is(err, wintun.ErrLoaded{}))
		require.True(t,
			err.(interface{ Temporary() bool }).Temporary(),
		)
	})

}

func Test_Open(t *testing.T) {
	t.Run("notload/open", func(t *testing.T) {
		ap, err := wintun.OpenAdapter("xxx")
		require.True(t, errors.Is(err, wintun.ErrNotLoad{}))
		require.Nil(t, ap)
	})
}
