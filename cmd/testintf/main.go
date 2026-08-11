package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/nokia/srlinux-ndk-go/ndk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	socketPath := "unix:///opt/srlinux/var/run/sr_sdk_service_manager:50053"

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			socketFile := strings.TrimPrefix(addr, "unix://")
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketFile)
		}),
	}

	conn, err := grpc.NewClient(socketPath, dialOpts...)
	if err != nil {
		fmt.Printf("Dial Error: %v\n", err)
		return
	}
	defer conn.Close()

	sdkClient := ndk.NewSdkMgrServiceClient(conn)
	notifClient := ndk.NewSdkNotificationServiceClient(conn)

	md := metadata.Pairs("agent_name", "srl_cyber_tui")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	regResp, err := sdkClient.AgentRegister(ctx, &ndk.AgentRegistrationRequest{
		AgentLiveliness: 10,
	})
	if err != nil {
		fmt.Printf("AgentRegister Error: %v\n", err)
		return
	}
	fmt.Printf("AgentRegister Success: AppId=%d\n", regResp.GetAppId())

	notifResp, err := sdkClient.NotificationRegister(ctx, &ndk.NotificationRegisterRequest{
		Op: ndk.NotificationRegisterRequest_Create,
		SubscriptionTypes: &ndk.NotificationRegisterRequest_Intf{
			Intf: &ndk.InterfaceSubscriptionRequest{
				Key: &ndk.InterfaceKey{IfName: "*"},
			},
		},
	})
	if err != nil {
		fmt.Printf("NotificationRegister Intf Error: %v\n", err)
		return
	}
	streamID := notifResp.GetStreamId()

	stream, err := notifClient.NotificationStream(ctx, &ndk.NotificationStreamRequest{
		StreamId: streamID,
	})
	if err != nil {
		fmt.Printf("NotificationStream Error: %v\n", err)
		return
	}

	fmt.Println("Streaming interface notifications...")
	for i := 0; i < 50; i++ {
		resp, err := stream.Recv()
		if err != nil {
			fmt.Printf("Stream Recv Error: %v\n", err)
			break
		}
		for _, n := range resp.GetNotification() {
			if intf := n.GetIntf(); intf != nil {
				key := intf.GetKey()
				data := intf.GetData()
				if key != nil && data != nil {
					fmt.Printf("Intf Notification: Key.IfName='%s' AdminIsUp=%d OperIsUp=%d Type=%v Description='%s'\n",
						key.GetIfName(), data.GetAdminIsUp(), data.GetOperIsUp(), data.GetIfType(), data.GetDescription())
				}
			}
		}
	}
}
