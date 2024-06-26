package wintun_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lysShub/wintun-go"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
	"gvisor.dev/gvisor/pkg/tcpip/header"
)

func Test_Invalid_Ring_Capacity(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	t.Run("lesser", func(t *testing.T) {
		ap, err := wintun.CreateAdapter("testinvalidringlesser")
		require.NoError(t, err)
		defer ap.Close()

		err = ap.Stop()
		require.NoError(t, err)

		err = ap.Start(wintun.MinRingCapacity - 1)
		require.Error(t, err)
	})
	t.Run("greater", func(t *testing.T) {
		ap, err := wintun.CreateAdapter("testinvalidringgreater")
		require.NoError(t, err)
		defer ap.Close()

		err = ap.Stop()
		require.NoError(t, err)

		err = ap.Start(wintun.MaxRingCapacity + 1)
		require.Error(t, err)
	})
}

func Test_Adapter_Create(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	t.Run("create/start", func(t *testing.T) {
		ap, err := wintun.CreateAdapter("createstart")
		require.NoError(t, err)
		defer ap.Close()

		err = ap.Start(wintun.MinRingCapacity)
		require.True(t, errors.Is(err, windows.ERROR_ALREADY_INITIALIZED))
	})
	t.Run("create/stop/stop", func(t *testing.T) {
		ap, err := wintun.CreateAdapter("createstopstop")
		require.NoError(t, err)
		defer ap.Close()

		err = ap.Stop()
		require.NoError(t, err)

		err = ap.Stop()
		require.NoError(t, err)
	})
	t.Run("create/close/close", func(t *testing.T) {
		ap, err := wintun.CreateAdapter("createcloseclose")
		require.NoError(t, err)

		err = ap.Close()
		require.NoError(t, err)

		err = ap.Close()
		require.NoError(t, err)
	})
}

func Test_Adapter_Stoped_Recv(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	ap, err := wintun.CreateAdapter("testadapterrwstoped")
	require.NoError(t, err)
	defer ap.Close()

	err = ap.Stop()
	require.NoError(t, err)

	_, err = ap.Recv(context.Background())
	require.Error(t, err)
}

func Test_Adapter_Index(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	name := "testadapterindex"

	a, err := wintun.CreateAdapter(name)
	require.NoError(t, err)
	defer a.Close()

	ifIdx, err := a.Index()
	require.NoError(t, err)

	b, err := exec.Command("netsh", "int", "ipv4", "show", "interfaces").CombinedOutput()
	require.NoError(t, err)

	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, name) {
			require.True(t, strings.Contains(line, strconv.Itoa(ifIdx)))
			return
		}
	}
	t.Errorf("can't found nic: \n %s", string(b))
}

func Test_Recv(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	var (
		ip    = netip.AddrFrom4([4]byte{10, 1, 1, 11})
		laddr = &net.UDPAddr{IP: ip.AsSlice(), Port: randPort()}
		raddr = &net.UDPAddr{IP: []byte{10, 1, 1, 13}, Port: randPort()}
	)

	ap, err := wintun.CreateAdapter("recvoutboundudp")
	require.NoError(t, err)
	defer ap.Close()

	luid, err := ap.GetAdapterLuid()
	require.NoError(t, err)
	addr := netip.PrefixFrom(ip, 24)
	err = luid.AddIPAddress(addr)
	require.NoError(t, err)

	// send udp packet
	var ret = make(chan struct{})
	msg := "fqwfnpina"
	go func() {
		conn, err := net.DialUDP("udp", laddr, raddr)
		require.NoError(t, err)
		for {
			select {
			case <-ret:
				return
			default:
			}

			n, err := conn.Write([]byte(msg))
			require.NoError(t, err)
			require.Equal(t, len(msg), n)

			time.Sleep(time.Second)
		}
	}()

	for {
		p, err := ap.Recv(context.Background())
		require.NoError(t, err)

		if header.IPVersion(p) == 4 {
			iphdr := header.IPv4(p)
			if iphdr.TransportProtocol() == header.UDPProtocolNumber {
				udphdr := header.UDP(iphdr.Payload())

				ok := iphdr.SourceAddress().As4() == laddr.AddrPort().Addr().As4() &&
					iphdr.DestinationAddress().As4() == raddr.AddrPort().Addr().As4() &&
					udphdr.SourcePort() == laddr.AddrPort().Port() &&
					udphdr.DestinationPort() == raddr.AddrPort().Port()

				if ok {
					require.Equal(t, string(udphdr.Payload()), msg)
					break
				}
			}
		}

		err = ap.Release(p)
		require.NoError(t, err)
	}
	ret <- struct{}{}
}

func Test_RecvCtx(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	ap, err := wintun.CreateAdapter("rcecvctx")
	require.NoError(t, err)
	defer ap.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for {
		p, err := ap.Recv(ctx)
		if err == nil {
			require.NoError(t, ap.Release(p))
		} else {
			require.True(t, errors.Is(err, context.DeadlineExceeded))
			return
		}
	}
}

func Test_Recving_Close(t *testing.T) {
	// if remove Close and Recv mutex, will fatal Exception
	wintun.MustLoad(wintun.DLL)

	for i := 0; i < 0xf; i++ {
		func() {
			ap, err := wintun.CreateAdapter("testrecvingclose")
			require.NoError(t, err)
			defer ap.Close()

			go func() {
				time.Sleep(time.Second)
				err := ap.Close()
				require.NoError(t, err)
			}()

			for {
				p, err := ap.Recv(context.Background())
				if err == nil {
					ap.Release(p)
				} else {
					require.True(t, errors.Is(err, wintun.ErrAdapterClosed{}))
					break
				}
			}
		}()
	}
}

func Test_Echo_UDP_Adapter(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	var (
		ip    = netip.AddrFrom4([4]byte{10, 0, 1, 3})
		laddr = &net.UDPAddr{IP: ip.AsSlice(), Port: randPort()}
		raddr = &net.UDPAddr{IP: []byte{10, 0, 1, 4}, Port: randPort()}
	)

	ap, err := wintun.CreateAdapter("testechoudpadapter")
	require.NoError(t, err)
	defer ap.Close()

	luid, err := ap.GetAdapterLuid()
	require.NoError(t, err)
	addr := netip.PrefixFrom(ip, 24)
	err = luid.SetIPAddresses([]netip.Prefix{addr})
	require.NoError(t, err)

	var ch = make(chan struct{})
	defer func() { <-ch }()
	go func() {
		defer close(ch)

		for {
			rp, err := ap.Recv(context.Background())
			if errors.Is(err, wintun.ErrAdapterClosed{}) {
				return
			}
			require.NoError(t, err)

			if header.IPVersion(rp) == 4 {
				iphdr := header.IPv4(rp)
				src := iphdr.SourceAddress()
				dst := iphdr.DestinationAddress()
				// not need update checksum
				iphdr.SetSourceAddress(dst)
				iphdr.SetDestinationAddress(src)

				if iphdr.TransportProtocol() == header.UDPProtocolNumber {
					udp := header.UDP(iphdr.Payload())
					src, dst := udp.SourcePort(), udp.DestinationPort()
					udp.SetSourcePort(dst)
					udp.SetDestinationPort(src)

					sp, _ := ap.Alloc(len(rp))
					copy(sp, rp)

					ap.Send(sp)
				}
			}
			ap.Release(rp)
		}
	}()

	conn, err := net.DialUDP("udp", laddr, raddr)
	require.NoError(t, err)
	defer conn.Close()

	msg := "fqwfnpina"
	n, err := conn.Write([]byte(msg))
	require.NoError(t, err)
	require.Equal(t, len(msg), n)

	var b = make([]byte, 1536)
	n, err = conn.Read(b)
	require.NoError(t, err)
	require.Equal(t, msg, string(b[:n]))

	require.NoError(t, ap.Close())
}

func Test_Packet_Sniffing(t *testing.T) {
	t.Skip("todo：maybe not route")
	// route add 0.0.0.0 mask 0.0.0.0 10.0.1.3 metric 5 if 116

	wintun.MustLoad(wintun.DLL)

	var (
		ip    = netip.AddrFrom4([4]byte{10, 0, 1, 3})
		laddr = &net.UDPAddr{IP: ip.AsSlice(), Port: randPort()}
		raddr = &net.UDPAddr{IP: []byte{8, 8, 8, 8}, Port: randPort()}
	)

	ap, err := wintun.CreateAdapter("testechoudpadapter")
	require.NoError(t, err)
	defer ap.Close()

	luid, err := ap.GetAdapterLuid()
	require.NoError(t, err)
	defer ap.Close()
	addr := netip.PrefixFrom(ip, 24)
	err = luid.AddIPAddress(addr)
	require.NoError(t, err)

	go func() {
		for {

			rp, err := ap.Recv(context.Background())
			require.NoError(t, err)

			if header.IPVersion(rp) == 4 {
				iphdr := header.IPv4(rp)
				src := iphdr.SourceAddress()
				dst := iphdr.DestinationAddress()

				fmt.Println(iphdr.TransportProtocol(), src.String(), "-->", dst.String())

				sp, err := ap.Alloc(len(rp))
				require.NoError(t, err)
				copy(sp, rp)
				err = ap.Send(sp)
				require.NoError(t, err)
			} else {
				fmt.Println("ipv6")
			}

			err = ap.Release(rp)
			require.NoError(t, err)
		}
	}()

	conn, err := net.DialUDP("udp", laddr, raddr)
	require.NoError(t, err)
	defer conn.Close()

	msg := "fqwfnpina"
	n, err := conn.Write([]byte(msg))
	require.NoError(t, err)
	require.Equal(t, len(msg), n)

	var b = make([]byte, 1536)
	n, err = conn.Read(b)
	require.NoError(t, err)
	require.Equal(t, msg, string(b[:n]))
}

func Test_Session_Restart(t *testing.T) {
	wintun.MustLoad(wintun.DLL)

	ap, err := wintun.CreateAdapter("testsessionrestart")
	require.NoError(t, err)
	defer ap.Close()

	err = ap.Stop()
	require.NoError(t, err)

	err = ap.Start(wintun.MinRingCapacity)
	require.NoError(t, err)
}

func Test_Auto_Handle_DF(t *testing.T) {
	t.Skip("todo")
}
