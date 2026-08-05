package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	organizationID := os.Args[1]
	conn, err := grpc.NewClient("127.0.0.1:15061", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := agentsv1.NewAgentsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-identity-id", os.Getenv("ADMIN_IDENTITY"))
	list, err := client.ListAgents(ctx, &agentsv1.ListAgentsRequest{OrganizationId: organizationID, PageSize: 200})
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	latest := map[string]string{
		"agent-init-agn":    "ghcr.io/agynio/agent-init-agn:0.5.18",
		"agent-init-codex":  "ghcr.io/agynio/agent-init-codex:0.13.47",
		"agent-init-claude": "ghcr.io/agynio/agent-init-claude:0.1.37",
	}
	for _, a := range list.GetAgents() {
		var want string
		for family, image := range latest {
			if strings.Contains(a.GetInitImage(), family) {
				want = image
			}
		}
		if want == "" || a.GetInitImage() == want {
			continue
		}
		if _, err := client.UpdateAgent(ctx, &agentsv1.UpdateAgentRequest{Id: a.GetMeta().GetId(), InitImage: &want}); err != nil {
			fmt.Printf("%s: ERROR %v\n", a.GetNickname(), err)
			continue
		}
		fmt.Printf("%s -> %s\n", a.GetNickname(), want)
	}
}
