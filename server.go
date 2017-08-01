package main

import (
	"net"
	"encoding/binary"
	"strings"
	"strconv"
	"errors"
	"fmt"
	"time"
	"sync"

	"github.com/golang/glog"
	"github.com/technofy/udp-loadbalancer-go/config"
)

type Connection struct {
	Target *net.UDPConn
	Client *net.UDPAddr
	Terminate chan bool

	LastClientActivity time.Time
}

type Server struct {
	wg sync.WaitGroup
	Terminate chan bool

	Config 	*config.Server
	Addr	*net.UDPAddr
	Conn	*net.UDPConn

	ConnectionPool map[uint32]Connection

	PassUpstream *Upstream
	PassHost *net.UDPAddr
	PassPort int
}


// FromIPPort is a quick helper to create an UDPAddr object
func FromIPPort(ip string, port int) *net.UDPAddr {
	return &net.UDPAddr{
		IP: net.ParseIP(ip),
		Port: port,
	}
}

// ListenTarget listens to incoming data returned by a target and bounces it back to the client
func (m *Server) ListenTarget(hash uint32, c Connection) {
	m.wg.Add(1)

	clearConn := func() {
		c.Target.Close()
		delete(m.ConnectionPool, hash)
		m.wg.Done()
	}

	// clear the connection in all cases
	defer clearConn()

	buf := make([]byte, 65536)

	timeoutDuration := 1 * time.Second
	for {
		select {
		case <-c.Terminate:
			return

		default:
		}

		c.Target.SetReadDeadline(time.Now().Add(timeoutDuration))
		n, err := c.Target.Read(buf)
		netErr, ok := err.(net.Error);

		if err == nil {
			m.Conn.WriteToUDP(buf[:n], c.Client)
			//glog.Info("Writing to client ", c.Client.IP.To4().String())
			continue
		}

		if ok && netErr.Timeout() {
			continue
		} else {
			glog.Error(err)
			return
		}
	}
}

func (m *Server) HandleClient(hash uint32, from *net.UDPAddr, to *net.UDPAddr, data []byte) error {
	if hash != 0 {
		conn, ok := m.ConnectionPool[hash]
		if !ok {
			rconn, err := net.DialUDP("udp4", nil, to)

			if err != nil {
				return err
			}

			conn = Connection{
				Target: rconn,
				Client: from,
				Terminate: make(chan bool, 1),
			}

			m.ConnectionPool[hash] = conn
			go m.ListenTarget(hash, conn)
		} else {
			// If the sending port has changed, change ours too
			if conn.Client.Port != from.Port {
				conn.Client.Port = from.Port
			}
		}

		conn.LastClientActivity = time.Now()
		//glog.Info("Writing to target: ", to.IP.To4().String())
		conn.Target.Write(data)
		m.ConnectionPool[hash] = conn
	} else {
		rconn, err := net.DialUDP("udp4", nil, to)
		if err != nil {
			return err
		}

		rconn.Write(data)
	}

	return nil
}

func (m *Server) LoadBalanceUDP() {
	m.wg.Add(1)

	defer m.Conn.Close()
	defer m.wg.Done()

	var targetAddr *net.UDPAddr
	var hash uint32

	buf := make([]byte, 65536)

	clientActivityTimeout := 1 * time.Minute
	timeoutDuration := 1 * time.Second

	for {
		select {
			case <- m.Terminate:
				return

			default:
		}

		//First check for dead connections
		for _, v := range m.ConnectionPool {
			if v.LastClientActivity.Before(time.Now().Add(-clientActivityTimeout)) {
				v.Terminate <- true
			}
		}

		m.Conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		n, addr, err := m.Conn.ReadFromUDP(buf)
		netErr, ok := err.(net.Error);

		if err == nil {
			if m.PassUpstream != nil {
				switch m.PassUpstream.HashType {
				case HT_REMOTE_IP:
					hash = binary.BigEndian.Uint32(addr.IP.To4())
					break

				case HT_REMOTE_PORT:
					hash = uint32(addr.Port)
					break

				default:
					hash = 0
				}

				peer, err := m.PassUpstream.GetPeer(hash)
				if err != nil {
					glog.Error(err)
					break
				}

				targetAddr = FromIPPort(*peer, m.PassPort)
			} else if m.PassHost != nil {
				targetAddr = m.PassHost
			} else {
				glog.Error("No static nor dynamic upstream, this shouldn't happen")
				break
			}

			//glog.Info("Handling client. IP: ", addr.IP.To4().String(), " Hash: ", fmt.Sprintf("%X", hash))
			err = m.HandleClient(hash, addr, targetAddr, buf[:n])
			if err != nil {
				glog.Error(err)
				break
			}
		} else {
			if ok && netErr.Timeout() {
				continue
			} else {
				glog.Error(err)
				return
			}
		}
	}
}

func (m *Server) Start() error {
	var err error

	m.Addr = FromIPPort(m.Config.Address, m.Config.Port)
	m.Conn, err = net.ListenUDP("udp", m.Addr)

	go m.LoadBalanceUDP()

	return err
}

func (m *Server) MustStart() {
	err := m.Start()
	if err != nil {
		panic(err)
	}
}

func (m *Server) Stop() {
	//Terminate connections gracefully
	for _, v := range m.ConnectionPool {
		v.Terminate <- true
	}

	m.Terminate <- true
	m.wg.Wait()
}

func NewServer(cfg *config.Server, upstreams []*Upstream) (*Server, error) {
	var passHost string

	server := &Server{
		Config: cfg,
		PassUpstream: nil,
		PassHost: nil,
		Terminate: make(chan bool, 1),
		ConnectionPool: make(map[uint32]Connection),
	}

	pass := cfg.Pass
	portSep := strings.LastIndex(pass, ":")

	if portSep == -1 {
		server.PassPort = cfg.Port
		passHost = pass
	} else {
		port, err := strconv.ParseInt(pass[portSep+1:], 10, 32)

		if err != nil {
			return nil, err
		}

		server.PassPort = int(port)
		passHost = pass[:portSep]
	}

	foundUpstream := false

	// Fill the Pass fields for the server
	for idx, _ := range upstreams {
		if strings.Compare(passHost, upstreams[idx].Config.Name) == 0 {
			//found an upstream
			server.PassUpstream = upstreams[idx]
			foundUpstream = true
			break
		}
	}

	if !foundUpstream {
		var err error
		server.PassHost, err = net.ResolveUDPAddr("udp4", passHost)

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Can't resolve upstream: %s", passHost))
		}
	}

	return server, nil
}

func MustNewServer(cfg *config.Server, upstreams []*Upstream) *Server {
	server, err := NewServer(cfg, upstreams)
	if err != nil {
		panic(err)
	}

	return server
}

