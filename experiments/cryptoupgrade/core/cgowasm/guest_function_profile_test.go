package cgowasm

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

type guestFunctionStat struct {
	name      string
	calls     int
	inclusive time.Duration
	exclusive time.Duration
}

type guestFunctionFrame struct {
	stat     *guestFunctionStat
	start    time.Time
	children time.Duration
}

type guestFunctionProfiler struct {
	stats  map[string]*guestFunctionStat
	frames []guestFunctionFrame
}

func (profiler *guestFunctionProfiler) Before(_ context.Context, _ api.Module, definition api.FunctionDefinition, _ []uint64, _ experimental.StackIterator) {
	name := definition.DebugName()
	stat := profiler.stats[name]
	profiler.frames = append(profiler.frames, guestFunctionFrame{stat: stat, start: time.Now()})
}

func (profiler *guestFunctionProfiler) After(_ context.Context, _ api.Module, _ api.FunctionDefinition, _ []uint64) {
	profiler.finish()
}

func (profiler *guestFunctionProfiler) Abort(_ context.Context, _ api.Module, _ api.FunctionDefinition, _ error) {
	profiler.finish()
}

func (profiler *guestFunctionProfiler) finish() {
	last := len(profiler.frames) - 1
	frame := profiler.frames[last]
	elapsed := time.Since(frame.start)
	profiler.frames = profiler.frames[:last]
	frame.stat.calls++
	frame.stat.inclusive += elapsed
	frame.stat.exclusive += elapsed - frame.children
	if last > 0 {
		profiler.frames[last-1].children += elapsed
	}
}

func profileGuestFunction(name string) bool {
	for _, marker := range []string{
		".execute", ".groth16_", ".mclBn_millerLoopVec", ".mclBn_finalExp",
		".mclBnG1_mul", ".mclBnG1_isValid", ".mclBnG2_isValid",
		".blst_miller_loop_n", ".blst_final_exp", ".blst_p1_mult",
		".blst_p1_affine_in_g1", ".blst_p2_affine_in_g2",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func TestGroth16GuestFunctionProfile(t *testing.T) {
	repoRoot, err := FindRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"groth16_bn254_verify", "groth16_bls12381_verify"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "zkgo")
			vectors, err := loadGroth16Inputs(root, name)
			if err != nil {
				t.Fatal(err)
			}
			input, err := packBytes3(vectors[0], vectors[1], vectors[2])
			if err != nil {
				t.Fatal(err)
			}
			wasm, err := os.ReadFile(filepath.Join(root, name, name+".wasm"))
			if err != nil {
				t.Fatal(err)
			}
			profiler := &guestFunctionProfiler{stats: make(map[string]*guestFunctionStat)}
			factory := experimental.FunctionListenerFactoryFunc(func(definition api.FunctionDefinition) experimental.FunctionListener {
				debugName := definition.DebugName()
				if !profileGuestFunction(debugName) {
					return nil
				}
				profiler.stats[debugName] = &guestFunctionStat{name: debugName}
				return profiler
			})
			ctx := experimental.WithFunctionListenerFactory(context.Background(), factory)
			verify, closeRuntime := prepareWASMVerificationWithContext(t, ctx, wasm, input, false)
			defer closeRuntime()
			if len(profiler.stats) == 0 {
				t.Fatal("WASM module has no matching named guest functions")
			}
			for sample := 0; sample < 3; sample++ {
				if err := verify(); err != nil {
					t.Fatal(err)
				}
			}
			if len(profiler.frames) != 0 {
				t.Fatal("unbalanced guest function callbacks")
			}
			for _, stat := range profiler.stats {
				stat.calls, stat.inclusive, stat.exclusive = 0, 0, 0
			}
			const samples = 10
			var total time.Duration
			for sample := 0; sample < samples; sample++ {
				start := time.Now()
				if err := verify(); err != nil {
					t.Fatal(err)
				}
				total += time.Since(start)
			}
			if len(profiler.frames) != 0 {
				t.Fatal("unbalanced guest function callbacks")
			}
			stats := make([]*guestFunctionStat, 0, len(profiler.stats))
			for _, stat := range profiler.stats {
				if stat.calls > 0 {
					stats = append(stats, stat)
				}
			}
			if len(stats) == 0 {
				t.Fatal("no matching guest functions called")
			}
			requiredFunctions := []string{".execute", "." + name, ".mclBn_millerLoopVec", ".mclBn_finalExp"}
			if name == "groth16_bls12381_verify" {
				requiredFunctions[2], requiredFunctions[3] = ".blst_miller_loop_n", ".blst_final_exp"
			}
			for _, required := range requiredFunctions {
				found := false
				for _, stat := range stats {
					if strings.HasSuffix(stat.name, required) && stat.calls == samples {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected %d calls to guest function %s", samples, required)
				}
			}
			sort.Slice(stats, func(left, right int) bool { return stats[left].inclusive > stats[right].inclusive })
			t.Logf("instrumented calls=%d mean=%s (not comparable with uninstrumented baseline)", samples, total/samples)
			for _, stat := range stats {
				t.Logf("%-64s calls=%5d inclusive=%10s excluding-selected-children=%10s", stat.name, stat.calls, stat.inclusive, stat.exclusive)
			}
		})
	}
}
