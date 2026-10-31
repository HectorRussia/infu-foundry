package contents

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestNewKeysLandAtTopWithoutRewritingTheGroup(t *testing.T) {
	var slots []OrderSlot
	previous := int64(1)
	for i := 0; i < 40; i++ {
		key, updates, err := KeyAtTop(slots)
		if err != nil {
			t.Fatal(err)
		}
		if len(updates) != 0 {
			t.Fatalf("insert %d rewrote %d rows", i, len(updates))
		}
		if i > 0 && key >= previous {
			t.Fatalf("key %d is not above %d", key, previous)
		}
		previous = key
		slots = append(slots, OrderSlot{ID: strconv.Itoa(i), Key: key})
		assertUniqueKeys(t, slots)
	}
}

func TestMoveWritesOnlyTheMovedRow(t *testing.T) {
	slots := []OrderSlot{
		{ID: "a", Key: positionGap},
		{ID: "b", Key: 2 * positionGap},
		{ID: "c", Key: 3 * positionGap},
	}
	updates, err := PlanMove(slots, "c", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates["c"] >= slots[0].Key {
		t.Fatalf("moving to the top must change only c, got %v", updates)
	}
	got := idsOf(applySlotKeys(slots, updates))
	if got != "c,a,b" {
		t.Fatalf("order %s", got)
	}

	after := "a"
	updates, err = PlanMove(slots, "c", &after)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 {
		t.Fatalf("moving between a wide gap must change one row, got %v", updates)
	}
	if updates["c"] <= slots[0].Key || updates["c"] >= slots[1].Key {
		t.Fatalf("c must sit strictly between a and b, got %d", updates["c"])
	}
	if _, touched := updates["a"]; touched {
		t.Fatal("a must keep its key")
	}
}

func TestMoveIsANoOpWhenAlreadyPlaced(t *testing.T) {
	slots := []OrderSlot{{ID: "a", Key: 10}, {ID: "b", Key: 20}}
	after := "a"
	updates, err := PlanMove(slots, "b", &after)
	if err != nil || len(updates) != 0 {
		t.Fatalf("already after a: %v %v", updates, err)
	}
	updates, err = PlanMove(slots, "a", nil)
	if err != nil || len(updates) != 0 {
		t.Fatalf("already at the top: %v %v", updates, err)
	}
}

func TestExhaustedGapRebalancesTheGroup(t *testing.T) {
	slots := []OrderSlot{{ID: "L", Key: 0}, {ID: "R", Key: 2}, {ID: "M", Key: 100}}
	after := "L"
	updates, err := PlanMove(slots, "M", &after)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates["M"] != 1 {
		t.Fatalf("one integer still fits, got %v", updates)
	}
	slots = applySlotKeys(slots, updates)
	updates, err = PlanMove(slots, "R", &after)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) < 2 {
		t.Fatalf("a gap of 1 must rebalance, got %v", updates)
	}
	final := applySlotKeys(slots, updates)
	assertUniqueKeys(t, final)
	if idsOf(final) != "L,R,M" {
		t.Fatalf("order %s", idsOf(final))
	}
}

func TestKeyAtTopRebalancesWhenTheLineIsFull(t *testing.T) {
	slots := []OrderSlot{{ID: "a", Key: math.MinInt64}}
	key, updates, err := KeyAtTop(slots)
	if err != nil {
		t.Fatal(err)
	}
	if key != 0 || updates["a"] != positionGap {
		t.Fatalf("key %d updates %v", key, updates)
	}
	if key == updates["a"] {
		t.Fatal("the new row collided with the rebalanced row")
	}
}

func TestPlanMoveRejectsAMissingNeighbour(t *testing.T) {
	slots := []OrderSlot{{ID: "a", Key: 1}}
	missing := "b"
	if _, err := PlanMove(slots, "a", &missing); err != ErrInvalid {
		t.Fatalf("neighbour: %v", err)
	}
	self := "a"
	if _, err := PlanMove(slots, "a", &self); err != ErrInvalid {
		t.Fatalf("self: %v", err)
	}
	if _, err := PlanMove(slots, "missing", nil); err != ErrNotFound {
		t.Fatalf("missing: %v", err)
	}
}

func TestFilteredListKeepsGlobalOrder(t *testing.T) {
	type row struct {
		id, category string
		pinned       bool
		key          int64
	}
	rows := []row{
		{id: "c", category: "A", key: 0},
		{id: "b", category: "B", key: positionGap},
		{id: "a", category: "A", key: 2 * positionGap},
		{id: "p", category: "A", pinned: true, key: 100},
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && Before(rows[j].pinned, rows[j].key, rows[j].id, rows[j-1].pinned, rows[j-1].key, rows[j-1].id); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	var filtered []string
	for _, row := range rows {
		if row.category == "A" {
			filtered = append(filtered, row.id)
		}
	}
	if ids := joinIDs(filtered); ids != "p,c,a" {
		t.Fatalf("category A must keep the global relative order, got %s", ids)
	}
}

func TestScratchKeysAvoidEveryOccupiedKey(t *testing.T) {
	existing := []int64{0, -1, -2, 5, math.MinInt64}
	scratch, err := ScratchKeys(existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(scratch) != len(existing) {
		t.Fatalf("got %v", scratch)
	}
	used := map[int64]bool{}
	for _, key := range existing {
		used[key] = true
	}
	for _, key := range scratch {
		if used[key] {
			t.Fatalf("scratch %d collides", key)
		}
		used[key] = true
	}
}

func TestConcurrentMovesKeepUniqueKeys(t *testing.T) {
	state := []OrderSlot{
		{ID: "a", Key: 0},
		{ID: "b", Key: positionGap},
		{ID: "c", Key: 2 * positionGap},
	}
	ids := []string{"a", "b", "c"}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for n := 0; n < 90; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := ids[n%len(ids)]
			var after *string
			if n%4 != 0 {
				other := ids[(n+1)%len(ids)]
				after = &other
			}
			mu.Lock()
			defer mu.Unlock()
			updates, err := PlanMove(state, id, after)
			if err != nil {
				t.Errorf("move %s: %v", id, err)
				return
			}
			if len(updates) == 1 {
				if _, ok := updates[id]; !ok {
					t.Errorf("single-row update was not the moved row: %v", updates)
				}
			}
			state = applySlotKeys(state, updates)
			assertUniqueKeys(t, state)
		}(n)
	}
	wg.Wait()
	if len(state) != 3 {
		t.Fatalf("lost a row: %v", state)
	}
}

func TestMigrationUsesTheSameLockAndGap(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "migrations", "20261031193000_add_content_order.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"20261031",
		"65536",
		"-- +goose Down",
		"DROP COLUMN IF EXISTS sort_key",
		"DROP COLUMN IF EXISTS pinned",
		"contents_live_sort_key",
		"EXECUTE FUNCTION contents_assign_sort_key()",
	} {
		if !bytes.Contains(body, []byte(want)) {
			t.Fatalf("migration missing %s", want)
		}
	}
}

func assertUniqueKeys(t *testing.T, slots []OrderSlot) {
	t.Helper()
	seen := map[int64]string{}
	for _, slot := range slots {
		if other, ok := seen[slot.Key]; ok {
			t.Fatalf("duplicate key %d on %s and %s", slot.Key, other, slot.ID)
		}
		seen[slot.Key] = slot.ID
	}
}

func idsOf(slots []OrderSlot) string {
	ids := make([]string, len(slots))
	for i, slot := range slots {
		ids[i] = slot.ID
	}
	return joinIDs(ids)
}

func joinIDs(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}
