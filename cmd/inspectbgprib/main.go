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

	// Query each potential child of /network-instance[name=default]/protocols/bgp
	children := []string{
		"export-policy", "import-policy", "as-path-options", "authentication",
		"best-path-selection", "bgp-label", "convergence", "dynamic-neighbors",
		"ebgp-default-policy", "failure-detection", "graceful-restart", "afi-safi",
		"preference", "rib-management", "route-advertisement", "route-flap-damping",
		"route-reflector", "transport", "trace-options", "statistics", "group",
		"neighbor", "admin-state", "oper-state", "under-maintenance",
		"maintenance-group", "autonomous-system", "local-preference", "router-id",
	}

	for _, child := range children {
		resp, err := gnmiClient.Get(ctx, &pb.GetRequest{
			Path: []*pb.Path{
				{
					Elem: []*pb.PathElem{
						{Name: "network-instance", Key: map[string]string{"name": "default"}},
						{Name: "protocols"},
						{Name: "bgp"},
						{Name: child},
					},
				},
			},
			Encoding: pb.Encoding_JSON_IETF,
		})
		if err != nil {
			continue
		}
		for _, n := range resp.GetNotification() {
			for _, u := range n.GetUpdate() {
				val := u.GetVal().GetJsonIetfVal()
				if len(val) > 2 {
					fmt.Printf("HAS DATA: /protocols/bgp/%s (bytes: %d)\n", child, len(val))
					var m map[string]interface{}
					if err := json.Unmarshal(val, &m); err == nil {
						findRouteKeys(m, "/protocols/bgp/"+child)
					}
				}
			}
		}
	}
}

func findRouteKeys(obj interface{}, path string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for k, val := range v {
			newPath := path + "/" + k
			if strings.Contains(strings.ToLower(k), "route") || strings.Contains(strings.ToLower(k), "evpn") || strings.Contains(strings.ToLower(k), "rib") {
				fmt.Printf("  -> SUBKEY: %s\n", newPath)
			}
			findRouteKeys(val, newPath)
		}
	case []interface{}:
		for i, val := range v {
			findRouteKeys(val, fmt.Sprintf("%s[%d]", path, i))
		}
	}
}
