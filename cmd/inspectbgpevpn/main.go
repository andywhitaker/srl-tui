package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func createMgmtDialer(socketPath string) func(ctx context.Context, addr string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		curNs, errCur := os.Open("/proc/self/ns/net")
		mgmtNs, errMgmt := os.Open("/run/netns/srbase-mgmt")

		if errMgmt == nil {
			_ = unix.Setns(int(mgmtNs.Fd()), unix.CLONE_NEWNET)
			mgmtNs.Close()
		}

		cleanPath := strings.TrimPrefix(socketPath, "unix://")
		var d net.Dialer
		conn, dialErr := d.DialContext(ctx, "unix", cleanPath)

		if errCur == nil {
			_ = unix.Setns(int(curNs.Fd()), unix.CLONE_NEWNET)
			curNs.Close()
		}

		return conn, dialErr
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

	client := pb.NewGNMIClient(conn)
	md := metadata.Pairs("username", "admin", "password", "NokiaSrl1!")
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), 5*time.Second)
	defer cancel()

	netInsts := []string{"default", "tenant1", "app", "web"}
	for _, inst := range netInsts {
		getResp, err := client.Get(ctx, &pb.GetRequest{
			Path: []*pb.Path{
				{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": inst}}, {Name: "protocols"}, {Name: "bgp"}, {Name: "rib-management"}}},
				{Elem: []*pb.PathElem{{Name: "network-instance", Key: map[string]string{"name": inst}}, {Name: "tunnel-table"}}},
			},
			Encoding: pb.Encoding_JSON_IETF,
		})
		if err != nil {
			continue
		}

		for _, n := range getResp.GetNotification() {
			for _, u := range n.GetUpdate() {
				val := string(u.GetVal().GetJsonIetfVal())
				var m map[string]interface{}
				_ = json.Unmarshal(u.GetVal().GetJsonIetfVal(), &m)
				if len(val) > 200 {
					val = val[:200] + "..."
				}
				fmt.Printf("NetInst [%s] Path: %v | Val: %s\n", inst, u.GetPath(), val)
			}
		}
	}
}
