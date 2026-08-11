package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"srl-tui/pkg/ndk"
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

	getResp, err := gnmiClient.Get(ctx, &pb.GetRequest{
		Path: []*pb.Path{
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "bridge-table"}, {Name: "mac-table"}}},
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "route-table"}}},
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "bgp-rib"}}},
		},
		Encoding: pb.Encoding_JSON_IETF,
	})

	if err == nil {
		for _, n := range getResp.GetNotification() {
			client.ParseGNMIStreamNotificationPublic(n)
		}
	}

	snap := state.Snapshot()

	type2Used, type3Used, type5Used := 0, 0, 0
	for _, r := range snap.EVPNRoutes {
		if r.Status == "u*>" {
			switch r.RouteType {
			case 2:
				type2Used++
			case 3:
				type3Used++
			case 5:
				type5Used++
			}
		}
	}

	fmt.Printf("SUMMARY CHECK:\n")
	fmt.Printf("  Type-2 Used Routes (u*>): %d (Expected: 24)\n", type2Used)
	fmt.Printf("  Type-3 Used Routes (u*>): %d (Expected: 6)\n", type3Used)
	fmt.Printf("  Type-5 Used Routes (u*>): %d (Expected: 6)\n", type5Used)

	fmt.Printf("\nSAMPLE ROUTES & RT FORMATS:\n")
	for i, r := range snap.EVPNRoutes {
		if r.Status == "u*>" {
			fmt.Printf("Route [%2d]: Type-%d RD=%-15s RT=%-15s VNI=%-15s MAC=%-18s IP=%-15s Paths=%d Neighbors=%s\n",
				i, r.RouteType, r.RD, r.RT, r.VNI, r.MAC, r.IP, len(r.PathVersions), r.Neighbor)
		}
	}
}
