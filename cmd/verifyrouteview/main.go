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
			{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": "*"}}, {Name: "route-table"}}},
		},
		Encoding: pb.Encoding_JSON_IETF,
	})

	if err == nil {
		for _, n := range getResp.GetNotification() {
			client.ParseGNMIStreamNotificationPublic(n)
		}
	} else {
		fmt.Printf("gNMI Get error: %v\n", err)
		return
	}

	snap := state.Snapshot()
	pal := theme.Cyberpunk
	rv := components.NewRouteView()

	fmt.Println("=== EMPIRICAL VERIFICATION: LIVE ROUTE TABLE VIEW ===")
	rendered := rv.Render(snap, pal, 130, 25, "", false, "")
	fmt.Println(rendered)

	fmt.Println("\n=== EMPIRICAL VERIFICATION: ROUTE DETAIL MODAL FOR FIRST ROUTE ===")
	if len(snap.RouteTable) > 0 {
		modal := components.RenderRouteDetailModal(snap.RouteTable[0], snap, pal, 130, 25)
		fmt.Println(modal)
	} else {
		fmt.Println("No routes returned!")
	}
}
