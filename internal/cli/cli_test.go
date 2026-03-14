package cli

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestExitCodeForDecision(t *testing.T) {
	if ExitCodeForDecision(policy.DecisionAllow) != ExitAllow {
		t.Errorf("allow should be %d, got %d", ExitAllow, ExitCodeForDecision(policy.DecisionAllow))
	}
	if ExitCodeForDecision(policy.DecisionWarn) != ExitWarn {
		t.Errorf("warn should be %d, got %d", ExitWarn, ExitCodeForDecision(policy.DecisionWarn))
	}
	if ExitCodeForDecision(policy.DecisionBlock) != ExitBlock {
		t.Errorf("block should be %d, got %d", ExitBlock, ExitCodeForDecision(policy.DecisionBlock))
	}
	if ExitCodeForDecision(policy.Decision("")) != ExitAllow {
		t.Errorf("unknown decision should map to allow")
	}
}

func TestExitCodeConstants(t *testing.T) {
	if ExitAllow != 0 {
		t.Errorf("ExitAllow must be 0 for CI")
	}
	if ExitError != 1 {
		t.Errorf("ExitError must be 1")
	}
	if ExitWarn != 2 {
		t.Errorf("ExitWarn must be 2")
	}
	if ExitBlock != 3 {
		t.Errorf("ExitBlock must be 3")
	}
}
