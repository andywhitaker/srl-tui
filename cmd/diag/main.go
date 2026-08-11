package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
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

func queryNeighbor(gnmiClient pb.GNMIClient, ctx context.Context, peerIP string) {
	getResp, err := gnmiClient.Get(ctx, &pb.GetRequest{
		Path: []*pb.Path{
			{
				Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "default"}},
					{Name: "protocols"},
					{Name: "bgp"},
					{Name: "neighbor", Key: map[string]string{"peer-address": peerIP}},
					{Name: "afi-safi"},
				},
			},
		},
		Encoding: pb.Encoding_JSON_IETF,
	})

	if err != nil {
		fmt.Printf("gNMI Get afi-safi error for %s: %v\n", peerIP, err)
		return
	}

	fmt.Printf("=== NEIGHBOR %s AFI-SAFI STATE ===\n", peerIP)
	for _, n := range getResp.GetNotification() {
		for _, u := range n.GetUpdate() {
			val := u.GetVal().GetJsonIetfVal()
			var pretty map[string]interface{}
			if err := json.Unmarshal(val, &pretty); err == nil {
				out, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Println(string(out))
			} else {
				fmt.Println(string(val))
			}
		}
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

	queryNeighbor(gnmiClient, ctx, "10.1.10.10")
	queryNeighbor(gnmiClient, ctx, "10.1.20.20")
}
