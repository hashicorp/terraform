package nomad

import (
	"testing"
	"time"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestPlanEndpoint_Submit(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)

	evalOut, token, err := s1.evalBroker.Dequeue([]string{eval1.Type}, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if evalOut != eval1 {
		t.Fatalf("Bad eval")
	}

	// Submit a plan
	plan := mock.Plan()
	plan.EvalID = eval1.ID
	plan.EvalToken = token
	req := &structs.PlanRequest{
		Plan:         plan,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.PlanResponse
	if err := msgpackrpc.CallWithCodec(codec, "Plan.Submit", req, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Result == nil {
		t.Fatalf("missing result")
	}
}
