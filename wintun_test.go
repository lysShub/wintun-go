package wintun_test

import (
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/lysShub/go-wintun"
	"github.com/lysShub/go-wintun/embed"
	"github.com/stretchr/testify/require"
)

func TestWintun(t *testing.T) {

	// err := dll.LoadDLL(dll.FileMode(`./embed/wintun_amd64.dll`))
	tun, err := wintun.LoadWintun(embed.Amd64)
	require.NoError(t, err)
	defer tun.Close()

	_, err = tun.DriverVersion()
	require.Equal(t, syscall.ERROR_FILE_NOT_FOUND, err)

	// guid := windows.GUID{
	// 	Data1: 0xdeadbabe,
	// 	Data2: 0xcafe,
	// 	Data3: 0xbeef,
	// 	Data4: [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
	// }

	a, err := tun.CreateAdapter("test", "example", nil)
	require.NoError(t, err)
	defer a.Close()

	v, err := tun.DriverVersion()
	require.NoError(t, err)
	require.NotZero(t, v)

	{
		ifIdx, err := a.InterfaceIndex()
		require.NoError(t, err)
		i, err := net.InterfaceByIndex(ifIdx)
		require.NoError(t, err)
		addrs, err := i.Addrs()
		require.NoError(t, err)
		require.Equal(t, 1, len(addrs))
		require.Equal(t, "ip+net", addrs[0].Network())
		fmt.Println(addrs[0].String())
	}

	s, err := a.StartSession(wintun.WINTUN_MIN_RING_CAPACITY)
	require.NoError(t, err)
	defer s.Close()

	{
		time.Sleep(time.Second * 20) // wait DHCP allocates IP addresses

		ifIdx, err := a.InterfaceIndex()
		require.NoError(t, err)
		i, err := net.InterfaceByIndex(ifIdx)
		require.NoError(t, err)
		addrs, err := i.Addrs()
		require.NoError(t, err)
		require.Equal(t, 2, len(addrs))
		// require.Equal(t, "ip", addrs[0].Network())
		fmt.Println(addrs[0].String())
		fmt.Println(addrs[1].String())
	}

	for {
		p, err := s.ReceivePacket()
		require.NoError(t, err)
		s.Release(p)
	}

}
