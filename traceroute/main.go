package traceroute

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func Traceroute(host string) {
	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		fmt.Println(err)
		return
	}

	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	for i := 1; i <= 64; i++ {
		// Create new ICMP Echo message
		b, err := (&icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID: os.Getpid() & 0xffff, Seq: i,
				Data: []byte("HELLO-R-U-THERE"),
			},
		}).Marshal(nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Set IP header's TTL field
		p := ipv4.NewPacketConn(c)
		if err := p.SetTTL(i); err != nil {
			fmt.Println(err)
			return
		}

		// Enable control message to return TTL
		if err := p.SetControlMessage(ipv4.FlagTTL, true); err != nil {
			fmt.Println(err)
			return
		}

		// Send the packet
		if _, err := c.WriteTo(b, dst); err != nil {
			fmt.Println(err)
			return
		}

		// Wait for a reply
		reply := make([]byte, 1500)
		err = c.SetReadDeadline(time.Now().Add(time.Second * 2))
		if err != nil {
			fmt.Println(err)
			return
		}
		n, peer, err := c.ReadFrom(reply)
		if err != nil {
			fmt.Println(err)
			continue
		}

		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
		if err != nil {
			fmt.Println(err)
			return
		}

		switch rm.Type {
		case ipv4.ICMPTypeTimeExceeded:
			// This is what we expect when the TTL is exceeded
			fmt.Printf("%d\t%s\n", i, peer)
		case ipv4.ICMPTypeEchoReply:
			// We got to the end
			fmt.Printf("%d\t%s\n", i, peer)
			return
		default:
			// Any other result is an error
			fmt.Printf("got %+v from %s; want ICMP time exceeded or echo reply", rm, peer)
		}
	}
}
