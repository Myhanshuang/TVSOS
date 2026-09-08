package gen

import (
	"testing"
	"time"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

func testInput(seed int64) Input {
	rules := sandboxpkg.DefaultWorldRules()
	// 提高事件概率，保证小样本也一定有事件产出
	rules.RandomEvents.WeatherProbPerTick = 0.01
	rules.RandomEvents.AccidentProbPerTick = 0.005
	return Input{
		ExperimentID:  99,
		Seed:          seed,
		DurationHours: 24,
		StartTime:     time.Date(2026, 1, 1, 8, 0, 0, 0, time.Local),
		TickSeconds:   10,
		Orders:        rules.Orders,
		Events:        rules.RandomEvents,
		VehicleLicenses: []string{"川A·T001", "川A·T002", "川A·T003"},
		PoiIDs:        []uint{1, 2, 3, 4, 5},
		CargoIDs:      []uint{10, 11},
	}
}

// TestGenerateDeterministic 同种子 ⇒ 订单流与事件脚本逐条一致（严格复现根基）。
func TestGenerateDeterministic(t *testing.T) {
	a := Generate(testInput(12345))
	b := Generate(testInput(12345))
	if len(a.Orders) != len(b.Orders) || len(a.Events) != len(b.Events) {
		t.Fatalf("same seed should produce same sizes, got orders %d/%d events %d/%d",
			len(a.Orders), len(b.Orders), len(a.Events), len(b.Events))
	}
	for i := range a.Orders {
		oa, ob := a.Orders[i], b.Orders[i]
		if !oa.VirtualTime.Equal(ob.VirtualTime) || oa.StartPoiId != ob.StartPoiId || oa.EndPoiId != ob.EndPoiId || oa.CargoId != ob.CargoId || oa.Count != ob.Count {
			t.Fatalf("order %d diverged with same seed", i)
		}
	}
	for i := range a.Events {
		ea, eb := a.Events[i], b.Events[i]
		if !ea.VirtualTime.Equal(eb.VirtualTime) || ea.EventType != eb.EventType || ea.VehicleLicense != eb.VehicleLicense || ea.DurationSeconds != eb.DurationSeconds {
			t.Fatalf("event %d diverged with same seed", i)
		}
	}
}

// TestGenerateSeedMatters 不同种子 ⇒ 订单流应不同。
func TestGenerateSeedMatters(t *testing.T) {
	a := Generate(testInput(12345))
	b := Generate(testInput(67890))
	if len(a.Orders) == 0 || len(b.Orders) == 0 {
		t.Fatal("orders should not be empty")
	}
	same := len(a.Orders) == len(b.Orders)
	if same {
		for i := range a.Orders {
			if !a.Orders[i].VirtualTime.Equal(b.Orders[i].VirtualTime) || a.Orders[i].StartPoiId != b.Orders[i].StartPoiId {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("different seeds should produce different order streams")
	}
}

// TestGenerateProducesEvents 提高概率后应产出天气与车祸脚本事件。
func TestGenerateProducesEvents(t *testing.T) {
	out := Generate(testInput(42))
	if len(out.Events) == 0 {
		t.Fatal("expected scripted events with raised probabilities")
	}
}

// TestGenerateDriverLevels 司机等级表：覆盖全部车辆、取值 2~5、同种子一致。
func TestGenerateDriverLevels(t *testing.T) {
	out := Generate(testInput(777))
	if len(out.DriverLevels) != 3 {
		t.Fatalf("expected 3 driver levels, got %d", len(out.DriverLevels))
	}
	for license, level := range out.DriverLevels {
		if level < 2 || level > 5 {
			t.Fatalf("level out of range for %s: %d", license, level)
		}
	}
	again := Generate(testInput(777))
	for license, level := range out.DriverLevels {
		if again.DriverLevels[license] != level {
			t.Fatalf("level diverged for %s with same seed", license)
		}
	}
}

// TestGenerateEmptyWithoutPois POI 不足时安全返回空订单流。
func TestGenerateEmptyWithoutPois(t *testing.T) {
	in := testInput(1)
	in.PoiIDs = []uint{1}
	out := Generate(in)
	if len(out.Orders) != 0 {
		t.Fatal("expected no orders with <2 POIs")
	}
}
