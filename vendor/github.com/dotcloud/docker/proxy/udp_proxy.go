package proxy

import (
	"encoding/binary"
	"github.com/dotcloud/docker/utils"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
)

const (
	UDPConnTrackTimeout = 90 * time.Second
	UDPBufSize          = 2048
)

// A net.Addr where the IP is split into two fields so you can use it as a key
// in a map:
type connTrackKey struct {
	IPHigh uint64
	IPLow  uint64
	Port   int
}

func newConnTrackKey(addr *net.UDPAddr) *connTrackKey {
	if len(addr.IP) == net.IPv4len {
		return &connTrackKey{
			IPHigh: 0,
			IPLow:  uint64(binary.BigEndian.Uint32(addr.IP)),
			Port:   addr.Port,
		}
	}
	return &connTrackKey{
		IPHigh: binary.BigEndian.Uint64(addr.IP[:8]),
		IPLow:  binary.BigEndian.Uint64(addr.IP[8:]),
		Port:   addr.Port,
	}
}

type connTrackMap map[connTrackKey]*net.UDPConn

type UDPProxy struct {
	listener       *net.UDPConn
	frontendAddr   *net.UDPAddr
	backendAddr    *net.UDPAddr
	connTrackTable connTrackMap
	connTrackLock  sync.Mutex
}

func NewUDPProxy(frontendAddr, backendAddr *net.UDPAddr) (*UDPProxy, error) {
	listener, err := net.ListenUDP("udp", frontendAddr)
	if err != nil {
		return nil, err
	}
	return &UDPProxy{
		listener:       listener,
		frontendAddr:   listener.LocalAddr().(*net.UDPAddr),
		backendAddr:    backendAddr,
		connTrackTable: make(connTrackMap),
	}, nil
}

func (proxy *UDPProxy) replyLoop(proxyConn *net.UDPConn, clientAddr *net.UDPAddr, clientKey *connTrackKey) {
	defer func() {
		proxy.connTrackLock.Lock()
		delete(proxy.connTrackTable, *clientKey)
		proxy.connTrackLock.Unlock()
		utils.Debugf("Done proxying between udp/%v and udp/%v", clientAddr.String(), proxy.backendAddr.String())
		proxyConn.Close()
	}()

	readBuf := make([]byte, UDPBufSize)
	for {
		proxyConn.SetReadDeadline(time.Now().Add(UDPConnTrackTimeout))
	again:
		read, err := proxyConn.Read(readBuf)
		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Err == syscall.ECONNREFUSED {
				// This will happen if the last write failed
				// (e.g: nothing is actually listening on the
				// proxied port on the container), ignore it
				// and continue until UDPConnTrackTimeout
				// expires:
				goto again
			}
			return
		}
		for i := 0; i != read; {
			written, err := proxy.listener.WriteToUDP(readBuf[i:read], clientAddr)
			if err != nil {
				return
			}
			i += written
			utils.Debugf("Forwarded %v/%v bytes to udp/%v", i, read, clientAddr.String())
		}
	}
}

func (proxy *UDPProxy) Run() {
	readBuf := make([]byte, UDPBufSize)
	utils.Debugf("Starting proxy on udp/%v for udp/%v", proxy.frontendAddr, proxy.backendAddr)
	for {
		read, from, err := proxy.listener.ReadFromUDP(readBuf)
		if err != nil {
			// NOTE: Apparently ReadFrom doesn't return
			// ECONNREFUSED like Read do (see comment in
			// UDPProxy.replyLoop)
			if utils.IsClosedError(err) {
				utils.Debugf("Stopping proxy on udp/%v for udp/%v (socket was closed)", proxy.frontendAddr, proxy.backendAddr)
			} else {
				utils.Errorf("Stopping proxy on udp/%v for udp/%v (%v)", proxy.frontendAddr, proxy.backendAddr, err.Error())
			}
			break
		}

		fromKey := newConnTrackKey(from)
		proxy.connTrackLock.Lock()
		proxyConn, hit := proxy.connTrackTable[*fromKey]
		if !hit {
			proxyConn, err = net.DialUDP("udp", nil, proxy.backendAddr)
			if err != nil {
				log.Printf("Can't proxy a datagram to udp/%s: %v\n", proxy.backendAddr.String(), err)
				continue
			}
			proxy.connTrackTable[*fromKey] = proxyConn
			go proxy.replyLoop(proxyConn, from, fromKey)
		}
		proxy.connTrackLock.Unlock()
		for i := 0; i != read; {
			written, err := proxyConn.Write(readBuf[i:read])
			if err != nil {
				log.Printf("Can't proxy a datagram to udp/%s: %v\n", proxy.backendAddr.String(), err)
				break
			}
			i += written
			utils.Debugf("Forwarded %v/%v bytes to udp/%v", i, read, proxy.backendAddr.String())
		}
	}
}

func (proxy *UDPProxy) Close() {
	proxy.listener.Close()
	proxy.connTrackLock.Lock()
	defer proxy.connTrackLock.Unlock()
	for _, conn := range proxy.connTrackTable {
		conn.Close()
	}
}

func (proxy *UDPProxy) FrontendAddr() net.Addr { return proxy.frontendAddr }
func (proxy *UDPProxy) BackendAddr() net.Addr  { return proxy.backendAddr }
