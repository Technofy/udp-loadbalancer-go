package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/technofy/udp-loadbalancer-go/config"
)


func main() {
	var upstreams []*Upstream
	var servers []*Server

	flag.Parse()

	glog.Info("Loading configuration file")
	settings, err := config.Load("config.yml")
	if err != nil {
		glog.Errorf("Can't read configuration file: %s\n", err.Error())
	}

	// Parse upstreams first
	// TODO: Kill AutoUpdate gracefully
	upstreams = make([]*Upstream, len(settings.Upstreams))
	for i := range settings.Upstreams {
		upstream := MustNewUpstream(&settings.Upstreams[i])

		if upstream.IsDynamic {
			go AutoUpdatePeer(upstream, 300)
		}

		upstreams[i] = upstream
	}

	// Then parse servers
	servers = make([]*Server, len(settings.Servers))
	for i := range settings.Servers {
		server, err := NewServer(&settings.Servers[i], upstreams)

		if err != nil {
			glog.Error(err)
			return
		}

		glog.Infof("Starting server on port: %d\n", server.Config.Port)
		server.MustStart()
		servers[i] = server
	}

	// Create the pacemaker for heartbeats
	pm, err := NewPacemakerAwsFromConfig(&settings.Pacemaker)
	if err != nil {
		glog.Warning(err)
		glog.Warning("Pacemaker error. Heartbeats disabled")
	} else {
		// Start sending heartbeats
		go pm.AutoHeartbeatAws(settings.Pacemaker.Interval)
	}

	// Wait for a termination signal
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	// Stop the service gracefully.
	for _, s := range servers {
		s.Stop()
	}
}