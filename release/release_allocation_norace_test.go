//go:build !race

package release

import "testing"

// TestReleasePlanAllocationBudgets is deliberately a non-race performance
// oracle. Race instrumentation adds bookkeeping allocations by design;
// functional validation and build-request derivation remain covered under
// -race by the release contract suites.
func TestReleasePlanAllocationBudgets(t *testing.T) {
	t.Parallel()
	plan := validReleasePlan(t)
	validation := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			if err := plan.Validate(); err != nil {
				b.Fatal(err)
			}
		}
	})
	if got := float64(validation.AllocsPerOp()); got > ReleasePlanValidateAllocationBudget {
		t.Fatalf("ReleasePlan.Validate allocs = %.0f, want <= %.0f", got, ReleasePlanValidateAllocationBudget)
	}

	build := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			if _, err := plan.GarbleBuildRequests(); err != nil {
				b.Fatal(err)
			}
		}
	})
	if got := float64(build.AllocsPerOp()); got > ReleasePlanBuildRequestAllocationBudget {
		t.Fatalf("ReleasePlan.GarbleBuildRequests allocs = %.0f, want <= %.0f", got, ReleasePlanBuildRequestAllocationBudget)
	}
}
