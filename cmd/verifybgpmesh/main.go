package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/components"
	"srl-tui/pkg/tui/theme"
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
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "default"}}, {Name: "protocols"}, {Name: "bgp"}}},
			{Elem: []*pb.PathElem{{Name: "system"}, {Name: "name"}}},
		},
		Encoding: pb.Encoding_JSON_IETF,
	})

	if err == nil {
		for _, n := range getResp.GetNotification() {
			client.ParseGNMIStreamNotificationPublic(n)
		}
	}

	snap := state.Snapshot()
	pal := theme.Cyberpunk

	fmt.Println("=== EMPIRICAL VERIFICATION: BGP PEER TABLE (PEER 0 SELECTED) ===")
	rendered0 := components.RenderTopoMesh(snap, true, pal, 130, 30, "", false, "", 0)
	fmt.Println(rendered0)

	fmt.Println("\n=== EMPIRICAL VERIFICATION: BGP PEER TABLE (PEER 1 SELECTED) ===")
	rendered1 := components.RenderTopoMesh(snap, true, pal, 130, 30, "", false, "", 1)
	fmt.Println(rendered1)
}
