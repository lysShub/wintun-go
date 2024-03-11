# wintun-go

golang client for [wintun](https://git.zx2c4.com/wintun/about/)


##### Example:
```golang
package main

import (
    "context"
    "log"
    "net"
    "net/netip"

    "github.com/lysShub/wintun-go"
    "golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
    "gvisor.dev/gvisor/pkg/tcpip/header" // go get gvisor.dev/gvisor@go
)

// curl google.com
func main() {
    wintun.MustLoad(wintun.DLL)
    defer wintun.Release()

    ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", "google.com")
    if err != nil {
        log.Fatal(err)
    }

    ap, err := wintun.CreateAdapter("capture-google")
    if err != nil {
        log.Fatal(err)
    }
    defer ap.Close()
    luid, err := ap.GetAdapterLuid()
    if err != nil {
        log.Fatal(err)
    }

    var addr = netip.PrefixFrom(netip.AddrFrom4([4]byte{10, 0, 7, 3}), 24)
    err = luid.SetIPAddresses([]netip.Prefix{addr})
    if err != nil {
        log.Fatal(err)
    }

    var routs []*winipcfg.RouteData
    for _, e := range ips {
        ip := netip.AddrFrom4([4]byte(e))
        dst := netip.PrefixFrom(ip, ip.BitLen())
        routs = append(routs, &winipcfg.RouteData{
            Destination: dst,
            NextHop:     addr.Addr(),
            Metric:      5,
        })
    }
    err = luid.AddRoutes(routs)
    if err != nil {
        log.Fatal(err)
    }

    for {
        ip, err := ap.Receive(context.Background())
        if err != nil {
            log.Fatal(err)
        }

        if header.IPVersion(ip) == 4 {
            iphdr := header.IPv4(ip)
            if iphdr.TransportProtocol() == header.TCPProtocolNumber {
                tcphdr := header.TCP(iphdr.Payload())

                log.Printf("%s:%d --> %s:%d %s\n",
                    iphdr.SourceAddress().String(), tcphdr.SourcePort(),
                    iphdr.DestinationAddress().String(), tcphdr.DestinationPort(),
                    tcphdr.Flags(),
                )
            }
        }

        err = ap.ReleasePacket(ip)
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

