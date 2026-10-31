package contents

import (
	"context"
	"errors"
	"math"
	"sort"

	"gorm.io/gorm"
)

const (
	// positionGap is the space left between neighbours so a later move can sit
	// between them by updating one row. Halving continues until the gap is 1.
	positionGap int64 = 65536
	// libraryOrderLock is the transaction advisory lock taken by every writer
	// that changes pinned or sort_key, including the insert trigger in
	// 20261031193000_add_content_order.sql. One lock covers both pin groups:
	// pinning and moving the same row must not interleave. It is an xact lock,
	// so it works through Neon's transaction-mode pooler.
	libraryOrderLock int64 = 20261031
)

// OrderSlot is one live content in a single pin group.
type OrderSlot struct {
	ID  string `gorm:"column:id"`
	Key int64  `gorm:"column:key"`
}

// Before reports whether a should appear earlier in the library than b.
// Pinned contents come first. Inside a group, smaller keys come first.
func Before(aPinned bool, aKey int64, aID string, bPinned bool, bKey int64, bID string) bool {
	if aPinned != bPinned {
		return aPinned
	}
	if aKey != bKey {
		return aKey < bKey
	}
	return aID < bID
}

// KeyAtTop is the sort key of a row placed at the top of a pin group.
// updates is empty unless the group had to be rebalanced first; those keys
// belong to the existing rows and must be written before the new key.
func KeyAtTop(slots []OrderSlot) (int64, map[string]int64, error) {
	slots = sortedSlots(slots)
	if key, fits := placeAtTop(lowestKey(slots)); fits {
		return key, nil, nil
	}
	updates := rebalancedKeys(slots)
	spaced := positionGap
	key, fits := placeAtTop(&spaced)
	if !fits {
		return 0, nil, ErrInvalid
	}
	return key, updates, nil
}

// PlanMove decides which sort keys must change to place id immediately after
// afterID. A nil afterID places it at the top of its group. The returned map
// is empty when the row is already there. It contains only id when an integer
// fits between the neighbours. It contains the whole group only when that gap
// is exhausted and the group is rebalanced.
func PlanMove(slots []OrderSlot, id string, afterID *string) (map[string]int64, error) {
	slots = sortedSlots(slots)
	idx := slotIndex(slots, id)
	if idx < 0 {
		return nil, ErrNotFound
	}
	if afterID != nil {
		if *afterID == id || slotIndex(slots, *afterID) < 0 {
			return nil, ErrInvalid
		}
	}
	if alreadyThere(slots, idx, afterID) {
		return map[string]int64{}, nil
	}
	if key, fits := placeRelative(slots, id, afterID); fits {
		if key == slots[idx].Key {
			return map[string]int64{}, nil
		}
		return map[string]int64{id: key}, nil
	}
	updates := rebalancedKeys(slots)
	next := applySlotKeys(slots, updates)
	key, fits := placeRelative(next, id, afterID)
	if !fits {
		return nil, ErrInvalid
	}
	if key == slots[idx].Key {
		delete(updates, id)
	} else {
		updates[id] = key
	}
	return updates, nil
}

// ScratchKeys returns one temporary key per existing key. None of them collide
// with existing or with each other, so a rebalance can park rows before
// writing the final keys without breaking the unique index.
func ScratchKeys(existing []int64) ([]int64, error) {
	used := make(map[int64]struct{}, len(existing))
	for _, key := range existing {
		used[key] = struct{}{}
	}
	out := make([]int64, 0, len(existing))
	candidate := int64(-1)
	for len(out) < len(existing) {
		if _, taken := used[candidate]; !taken {
			used[candidate] = struct{}{}
			out = append(out, candidate)
		}
		if candidate == math.MinInt64 {
			break
		}
		candidate--
	}
	if len(out) < len(existing) {
		return nil, ErrInvalid
	}
	return out, nil
}

func sortedSlots(slots []OrderSlot) []OrderSlot {
	out := append([]OrderSlot(nil), slots...)
	sort.Slice(out, func(i, j int) bool {
		return Before(false, out[i].Key, out[i].ID, false, out[j].Key, out[j].ID)
	})
	return out
}

func slotIndex(slots []OrderSlot, id string) int {
	for i, slot := range slots {
		if slot.ID == id {
			return i
		}
	}
	return -1
}

func alreadyThere(slots []OrderSlot, idx int, afterID *string) bool {
	var pred string
	hasPred := idx > 0
	if hasPred {
		pred = slots[idx-1].ID
	}
	if afterID == nil {
		return !hasPred
	}
	return hasPred && pred == *afterID
}

func lowestKey(slots []OrderSlot) *int64 {
	if len(slots) == 0 {
		return nil
	}
	key := slots[0].Key
	return &key
}

func lowestExcept(slots []OrderSlot, skipID string) (int64, bool) {
	found := false
	var min int64
	for _, slot := range slots {
		if slot.ID == skipID {
			continue
		}
		if !found || slot.Key < min {
			min = slot.Key
			found = true
		}
	}
	return min, found
}

func placeAtTop(min *int64) (int64, bool) {
	if min == nil {
		return 0, true
	}
	if *min < math.MinInt64+positionGap {
		return 0, false
	}
	return *min - positionGap, true
}

func placeAfter(prev int64, next *int64) (int64, bool) {
	if next == nil {
		if prev > math.MaxInt64-positionGap {
			return 0, false
		}
		return prev + positionGap, true
	}
	if *next <= prev {
		return 0, false
	}
	gap := *next - prev
	if gap < 2 {
		return 0, false
	}
	return prev + gap/2, true
}

func placeRelative(slots []OrderSlot, id string, afterID *string) (int64, bool) {
	if afterID == nil {
		min, found := lowestExcept(slots, id)
		if !found {
			return placeAtTop(nil)
		}
		return placeAtTop(&min)
	}
	prev, found := int64(0), false
	for _, slot := range slots {
		if slot.ID == *afterID {
			prev = slot.Key
			found = true
			break
		}
	}
	if !found {
		return 0, false
	}
	var next *int64
	for _, slot := range slots {
		if slot.ID == id || slot.ID == *afterID {
			continue
		}
		if slot.Key > prev {
			key := slot.Key
			next = &key
			break
		}
	}
	return placeAfter(prev, next)
}

func rebalancedKeys(slots []OrderSlot) map[string]int64 {
	updates := map[string]int64{}
	for i, slot := range slots {
		key := int64(i+1) * positionGap
		if key != slot.Key {
			updates[slot.ID] = key
		}
	}
	return updates
}

func applySlotKeys(slots []OrderSlot, updates map[string]int64) []OrderSlot {
	out := append([]OrderSlot(nil), slots...)
	for i, slot := range out {
		if key, ok := updates[slot.ID]; ok {
			out[i].Key = key
		}
	}
	return sortedSlots(out)
}

func lockLibraryOrder(tx *gorm.DB) error {
	// Give up rather than hold an HTTP request until the write timeout if
	// another transaction is stuck inside this lock.
	if err := tx.Exec(`SET LOCAL lock_timeout = '4s'`).Error; err != nil {
		return err
	}
	return tx.Exec(`SELECT pg_advisory_xact_lock(?)`, libraryOrderLock).Error
}

func groupSlots(tx *gorm.DB, pinned bool) ([]OrderSlot, error) {
	var slots []OrderSlot
	err := tx.Raw(`
SELECT id, sort_key AS key
FROM contents
WHERE deleted_at IS NULL AND pinned = ?
ORDER BY sort_key ASC, id ASC`, pinned).Scan(&slots).Error
	return slots, err
}

type placement struct {
	Pinned  bool  `gorm:"column:pinned"`
	SortKey int64 `gorm:"column:sort_key"`
}

func livePlacement(tx *gorm.DB, id string) (placement, error) {
	var rows []placement
	err := tx.Raw(
		`SELECT pinned, sort_key FROM contents WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(&rows).Error
	if err != nil {
		return placement{}, err
	}
	if len(rows) == 0 {
		return placement{}, ErrNotFound
	}
	return rows[0], nil
}

func trashedPlacement(tx *gorm.DB, id string) (placement, error) {
	var rows []placement
	err := tx.Raw(
		`SELECT pinned, sort_key FROM contents WHERE id = ? AND deleted_at IS NOT NULL`, id,
	).Scan(&rows).Error
	if err != nil {
		return placement{}, err
	}
	if len(rows) == 0 {
		return placement{}, ErrNotFound
	}
	return rows[0], nil
}

// applySortKeys writes the planned keys. A single-row move updates that row
// only. A rebalance parks the affected rows on temporary keys first so the
// unique (pinned, sort_key) index never sees a duplicate mid-update.
func applySortKeys(tx *gorm.DB, slots []OrderSlot, updates map[string]int64) error {
	if len(updates) == 0 {
		return nil
	}
	ids := make([]string, 0, len(updates))
	for id := range updates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 1 {
		return tx.Exec(
			`UPDATE contents SET sort_key = ? WHERE id = ? AND deleted_at IS NULL`,
			updates[ids[0]], ids[0],
		).Error
	}
	occupied := make([]int64, len(slots))
	for i, slot := range slots {
		occupied[i] = slot.Key
	}
	scratch, err := ScratchKeys(occupied)
	if err != nil {
		return err
	}
	if len(scratch) < len(ids) {
		return ErrInvalid
	}
	for i, id := range ids {
		if err := tx.Exec(
			`UPDATE contents SET sort_key = ? WHERE id = ? AND deleted_at IS NULL`,
			scratch[i], id,
		).Error; err != nil {
			return err
		}
	}
	for _, id := range ids {
		if err := tx.Exec(
			`UPDATE contents SET sort_key = ? WHERE id = ? AND deleted_at IS NULL`,
			updates[id], id,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// SetPinned moves a live content to the top of the pinned or unpinned group.
// Pinning and unpinning are shared by every account.
func (s *Store) SetPinned(ctx context.Context, id string, pinned bool) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLibraryOrder(tx); err != nil {
			return err
		}
		item, err := livePlacement(tx, id)
		if err != nil {
			return err
		}
		if item.Pinned == pinned {
			return nil
		}
		slots, err := groupSlots(tx, pinned)
		if err != nil {
			return err
		}
		key, updates, err := KeyAtTop(slots)
		if err != nil {
			return err
		}
		if err := applySortKeys(tx, slots, updates); err != nil {
			return err
		}
		result := tx.Exec(
			`UPDATE contents SET pinned = ?, sort_key = ? WHERE id = ? AND deleted_at IS NULL`,
			pinned, key, id,
		)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Move places a live content immediately after afterID in the same pin group.
// A nil afterID places it at the top of that group. Neighbours in the other
// group, or in the trash, are rejected.
func (s *Store) Move(ctx context.Context, id string, afterID *string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLibraryOrder(tx); err != nil {
			return err
		}
		item, err := livePlacement(tx, id)
		if err != nil {
			return err
		}
		if afterID != nil {
			if *afterID == id {
				return ErrInvalid
			}
			other, err := livePlacement(tx, *afterID)
			if errors.Is(err, ErrNotFound) {
				return ErrInvalid
			}
			if err != nil {
				return err
			}
			if other.Pinned != item.Pinned {
				return ErrInvalid
			}
		}
		slots, err := groupSlots(tx, item.Pinned)
		if err != nil {
			return err
		}
		updates, err := PlanMove(slots, id, afterID)
		if err != nil {
			return err
		}
		if err := applySortKeys(tx, slots, updates); err != nil {
			return err
		}
		_, err = livePlacement(tx, id)
		return err
	})
}
