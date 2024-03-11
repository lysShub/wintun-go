package wintun_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lysShub/wintun-go"
	"github.com/stretchr/testify/require"
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

func Test_DriverVersion(t *testing.T) {
	t.Skip("can't get")

	require.NoError(t, wintun.Load(wintun.DLL))
	defer wintun.Release()

	ver, err := wintun.DriverVersion()
	require.NoError(t, err)
	t.Log(ver)
}

func Test_Logger(t *testing.T) {
	require.NoError(t, wintun.Load(wintun.DLL))
	defer wintun.Release()

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
}

func Test_Adapter_Index(t *testing.T) {
	require.NoError(t, wintun.Load(wintun.DLL))
	defer wintun.Release()

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

func Test_Example(t *testing.T) {
	// https://github.com/WireGuard/wintun/blob/master/example/example.c

	require.NoError(t, wintun.Load(wintun.DLL))
	defer wintun.Release()

	ap, err := wintun.CreateAdapter("testexample")
	require.NoError(t, err)
	defer ap.Close()

	luid, err := ap.GetAdapterLuid()
	require.NoError(t, err)
	err = luid.SetIPAddresses([]netip.Prefix{
		netip.PrefixFrom(netip.AddrFrom4([4]byte{10, 6, 7, 7}), 24), // 10.6.7.7/24
	})
	require.NoError(t, err)

	// Send  ping -S 10.6.7.8 10.6.7.7
	go func() {
		for {
			p, err := ap.AllocPacket(28)
			require.NoError(t, err)

			{ // build ICMP Echo
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
					SrcAddr:        tcpip.AddrFrom4([4]byte{10, 6, 7, 8}), /* 10.6.7.8 */
					DstAddr:        tcpip.AddrFrom4([4]byte{10, 6, 7, 7}), /* 10.6.7.7 */
				})
				iphdr.SetChecksum(^checksum.Checksum(p[:iphdr.HeaderLength()], 0))
				require.True(t, iphdr.IsChecksumValid())

				icmphdr := header.ICMPv4(iphdr.Payload())
				icmphdr.SetType(header.ICMPv4Echo)
				icmphdr.SetChecksum(^checksum.Checksum(icmphdr, 0))
			}

			err = ap.Send(p)
			require.NoError(t, err)

			time.Sleep(time.Second)
		}
	}()

	for { // Receive outboud ICMP Echo-Reply packet
		p, err := ap.Receive(context.Background())
		require.NoError(t, err)

		var str string
		switch header.IPVersion(p) {
		case 4:
			iphdr := header.IPv4(p)
			if iphdr.TransportProtocol() == header.ICMPv4ProtocolNumber {
				icmphdr := header.ICMPv4(iphdr.Payload())

				str = fmt.Sprintf(
					"Received IPv%d proto 0x%x packet from %s to %s, icmp type %d",
					4, iphdr.TransportProtocol(), iphdr.SourceAddress(), iphdr.DestinationAddress(), icmphdr.Type(),
				)
			}
		default:
		}
		ap.ReleasePacket(p)

		if len(str) > 0 {
			// t.Log(str)
			return
		}
	}

}

func Test_Wintun_Recv(t *testing.T) {
	require.NoError(t, wintun.Load(wintun.DLL))
	defer wintun.Release()

	t.Run("recv-outbound-udp", func(t *testing.T) {
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
		err = luid.SetIPAddresses([]netip.Prefix{addr})
		require.NoError(t, err)

		// send udp packet
		msg := "fqwfnpina"
		go func() {
			conn, err := net.DialUDP("udp", laddr, raddr)
			require.NoError(t, err)
			for {
				n, err := conn.Write([]byte(msg))
				require.NoError(t, err)
				require.Equal(t, len(msg), n)

				time.Sleep(time.Second)
			}
		}()

		for {
			p, err := ap.Receive(context.Background())
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
						return
					}
				}
			}

			err = ap.ReleasePacket(p)
			require.NoError(t, err)
		}
	})

	t.Run("recv-ctx", func(t *testing.T) {
		ap, err := wintun.CreateAdapter("cecvctx")
		require.NoError(t, err)
		defer ap.Close()

		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		for {
			p, err := ap.Receive(ctx)
			if p != nil {
				require.NoError(t, ap.ReleasePacket(p))
			} else {
				require.True(t, errors.Is(err, context.DeadlineExceeded))
				return
			}
		}
	})
}
