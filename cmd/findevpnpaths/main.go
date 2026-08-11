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
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), 10*time.Second)
	defer cancel()

	// Query /network-instance[name=default]/protocols/bgp
	getResp, err := gnmiClient.Get(ctx, &pb.GetRequest{
		Path: []*pb.Path{
			{
				Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "default"}},
					{Name: "protocols"},
					{Name: "bgp"},
				},
			},
		},
		Encoding: pb.Encoding_JSON_IETF,
	})

	if err != nil {
		fmt.Printf("gNMI Get BGP error: %v\n", err)
		return
	}

	for _, n := range getResp.GetNotification() {
		for _, u := range n.GetUpdate() {
			val := u.GetVal().GetJsonIetfVal()
			var data map[string]interface{}
			if err := json.Unmarshal(val, &data); err == nil {
				// Search for keys related to evpn or routes or rib
				dumpKeys(data, "")
			}
		}
	}
}

func dumpKeys(obj interface{}, prefix string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for k, val := range v {
			newPfx := prefix + "/" + k
			if strings.Contains(strings.ToLower(k), "evpn") || strings.Contains(strings.ToLower(k), "route") || strings.Contains(strings.ToLower(k), "rib") {
				fmt.Printf("FOUND KEY: %s\n", newPfx)
			}
			dumpKeys(val, newPfx)
		}
	case []interface{}:
		for i, val := range v {
			dumpKeys(val, fmt.Sprintf("%s[%d]", prefix, i))
		}
	}
}
