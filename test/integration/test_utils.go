package integration

import (
	"context"
	"fmt"
	"os/exec"

	v2beta2 "github.com/weaveworks/weave-policy-agent/api/v2beta2"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func kubectl(args ...string) error {
	cmd := exec.Command("kubectl", args...)
	stdout, err := cmd.Output()
	fmt.Println(string(stdout))
	return err
}

func listPolicies(ctx context.Context, c client.Client) (*v2beta2.PolicyList, error) {
	var policies v2beta2.PolicyList
	if err := c.List(ctx, &policies); err != nil {
		return nil, err
	}
	return &policies, nil
}

func listViolationEvents(ctx context.Context, c client.Client, opts []client.ListOption) (*v1.EventList, error) {
	var events v1.EventList
	if err := c.List(ctx, &events, opts...); err != nil {
		return nil, err
	}
	return &events, nil
}
