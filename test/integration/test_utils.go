package integration

import (
	"context"
	"fmt"
	"os/exec"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func kubectl(args ...string) error {
	cmd := exec.Command("kubectl", args...)
	stdout, err := cmd.Output()
	fmt.Println(string(stdout))
	return err
}

func listViolationEvents(ctx context.Context, c client.Client, opts []client.ListOption) (*v1.EventList, error) {
	var events v1.EventList
	if err := c.List(ctx, &events, opts...); err != nil {
		return nil, err
	}
	return &events, nil
}
