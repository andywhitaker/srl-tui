package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/components"
	"strings"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func createMgmtDialer(socketPath string) func(ctx context.Context, addr string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		cleanPath := strings.TrimPrefix(socketPath, "unix://")
		var d net.Dialer
		return d.DialContext(ctx, "unix", cleanPath)
	}
}

func main() {
	sock := "unix:///opt/srlinux/var/run/sr_grpc_server_insecure-mgmt"
	conn, err := grpc.NewClient(sock,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(createMgmtDialer(sock)),
	)
	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
		return
	}
	defer conn.Close()

	gnmiClient := pb.NewGNMIClient(conn)
	state := ndk.NewTelemetryState(16)
	client := ndk.NewNDKClient(sock, state)

	user := os.Getenv("SRL_USERNAME")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("SRL_PASSWORD")
	if pass == "" {
		pass = "NokiaSrl1!"
	}
	md := metadata.Pairs("username", user, "password", pass)
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), 5*time.Second)
	defer cancel()

	// Query MAC table, route table, BGP RIB
	getResp, err := gnmiClient.Get(ctx, &pb.GetRequest{
		Path: []*pb.Path{
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "bridge-table"}, {Name: "mac-table"}}},
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "route-table"}}},
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "bgp-rib"}}},
		},
		Encoding: pb.Encoding_JSON_IETF,
	})

	if err == nil {
		fmt.Printf("Get returned %d notifications\n", len(getResp.GetNotification()))
		for _, n := range getResp.GetNotification() {
			client.ParseGNMIStreamNotificationPublic(n)
		}
	} else {
		fmt.Printf("gNMI Get error: %v\n", err)
	}

	snap := state.Snapshot()
	fmt.Printf("SNAPSHOT EVPN ROUTES (TOTAL=%d):\n", len(snap.EVPNRoutes))

	filteredDefault := components.GetFilteredEVPNRoutes(snap, components.EVPNFilterAll, "", false)
	fmt.Printf("\nFILTERED DEFAULT (showUnimported=false) TOTAL: %d\n", len(filteredDefault))
	for i, r := range filteredDefault {
		fmt.Printf("Default [%2d]: Type-%d RD=%-15s VNI=%-15s MAC=%-18s IP=%-15s Status=%s\n",
			i, r.RouteType, r.RD, r.VNI, r.MAC, r.IP, r.Status)
	}

	filteredAll := components.GetFilteredEVPNRoutes(snap, components.EVPNFilterAll, "", true)
	fmt.Printf("\nFILTERED ALL (showUnimported=true) TOTAL: %d\n", len(filteredAll))
	for i, r := range filteredAll {
		fmt.Printf("All     [%2d]: Type-%d RD=%-15s VNI=%-15s MAC=%-18s IP=%-15s Status=%s\n",
			i, r.RouteType, r.RD, r.VNI, r.MAC, r.IP, r.Status)
	}
}
