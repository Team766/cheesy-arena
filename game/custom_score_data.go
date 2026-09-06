//go:build custom

package game

import (
	"encoding/json"
)

type Phase int

const (
	PhaseAuto Phase = iota
	PhaseTeleop
	PhaseEndgame
)

// PhaseFromString maps a YAML phase name onto its Phase constant.
func PhaseFromString(phase string) (Phase, bool) {
	switch phase {
	case "auto":
		return PhaseAuto, true
	case "teleop":
		return PhaseTeleop, true
	case "endgame":
		return PhaseEndgame, true
	}
	return PhaseAuto, false
}

func (p Phase) String() string {
	switch p {
	case PhaseAuto:
		return "auto"
	case PhaseTeleop:
		return "teleop"
	case PhaseEndgame:
		return "endgame"
	}
	return ""
}

// CustomScoreData and Score deliberately carry no json tags: the JSON keys are the Go field names
// (Counts, BoolStatuses, EnumStatuses, Fouls, PlayoffDq, Hub), matching the stock build's
// convention and the keys the web UI reads. game/custom_score_data_test.go pins the exact key set.
type CustomScoreData struct {
	Counts       map[string]int
	BoolStatuses map[string][3]bool
	EnumStatuses map[string][3]int
}

type Score struct {
	CustomScoreData
	Fouls     []Foul
	PlayoffDq bool
	Hub       Hub
}

func (s *Score) ensureInit() {
	if s.Counts == nil {
		s.Counts = make(map[string]int)
	}
	if s.BoolStatuses == nil {
		s.BoolStatuses = make(map[string][3]bool)
	}
	if s.EnumStatuses == nil {
		s.EnumStatuses = make(map[string][3]int)
	}
}

func (s *Score) UnmarshalJSON(data []byte) error {
	type Alias Score
	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*s = Score(*aux)
	s.ensureInit()
	return nil
}

func (s *Score) GetCount(id string, phase Phase) int {
	s.ensureInit()
	key := id + "_" + phase.String()
	return s.Counts[key]
}

// AdjustCount adds delta to the count for the given scoring element and phase, clamping at zero. It
// returns true if the stored value changed. An unknown element id, an element that is not scored in
// the given phase, or the absence of an active config all cause the mutation to be rejected.
func (s *Score) AdjustCount(id string, phase Phase, delta int) bool {
	count := GetActiveConfig().Count(id)
	if count == nil {
		return false
	}
	valid := false
	for _, pp := range count.Phases {
		if pp.Phase == phase.String() {
			valid = true
			break
		}
	}
	if !valid {
		return false
	}

	s.ensureInit()
	key := id + "_" + phase.String()
	curr := s.Counts[key]
	newVal := curr + delta
	if newVal < 0 {
		newVal = 0
	}
	if newVal == curr {
		return false
	}
	s.Counts[key] = newVal
	return true
}

// SetBoolStatus sets a bool status for one robot, returning true if the stored value changed. An
// unknown id, an enum status, an out-of-range robot index, or the absence of an active config all
// cause the mutation to be rejected.
func (s *Score) SetBoolStatus(id string, robotIndex int, value bool) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	status := GetActiveConfig().Status(id)
	if status == nil || len(status.Values) > 0 {
		return false
	}

	s.ensureInit()
	arr := s.BoolStatuses[id]
	if arr[robotIndex] == value {
		return false
	}
	arr[robotIndex] = value
	s.BoolStatuses[id] = arr
	return true
}

func (s *Score) GetBoolStatus(id string, robotIndex int) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	s.ensureInit()
	return s.BoolStatuses[id][robotIndex]
}

// SetEnumStatus sets an enum status for one robot to the value at the given index, returning true
// if the stored value changed. An unknown id, a bool status, an out-of-range value or robot index,
// or the absence of an active config all cause the mutation to be rejected.
func (s *Score) SetEnumStatus(id string, robotIndex int, valueIndex int) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	status := GetActiveConfig().Status(id)
	if status == nil || len(status.Values) == 0 {
		return false
	}
	if valueIndex < 0 || valueIndex >= len(status.Values) {
		return false
	}

	s.ensureInit()
	arr := s.EnumStatuses[id]
	if arr[robotIndex] == valueIndex {
		return false
	}
	arr[robotIndex] = valueIndex
	s.EnumStatuses[id] = arr
	return true
}

func (s *Score) SetEnumStatusByID(id string, robotIndex int, valueId string) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	status := GetActiveConfig().Status(id)
	if status == nil {
		return false
	}
	for idx, v := range status.Values {
		if v.ID == valueId {
			return s.SetEnumStatus(id, robotIndex, idx)
		}
	}
	return false
}

func (s *Score) GetEnumStatus(id string, robotIndex int) int {
	if robotIndex < 0 || robotIndex >= 3 {
		return 0
	}
	s.ensureInit()
	return s.EnumStatuses[id][robotIndex]
}

// CycleEnumStatus advances an enum status for one robot to its next value, wrapping around. It
// returns false for an unknown id, a bool status, an out-of-range robot index, or when no config is
// active.
func (s *Score) CycleEnumStatus(id string, robotIndex int) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	status := GetActiveConfig().Status(id)
	if status == nil || len(status.Values) == 0 {
		return false
	}

	curr := s.GetEnumStatus(id, robotIndex)
	next := (curr + 1) % len(status.Values)
	return s.SetEnumStatus(id, robotIndex, next)
}

func (s *Score) AnyBoolStatus(id string) bool {
	s.ensureInit()
	arr, ok := s.BoolStatuses[id]
	if !ok {
		return false
	}
	for _, v := range arr {
		if v {
			return true
		}
	}
	return false
}

func (s *Score) CountBoolStatus(id string) int {
	s.ensureInit()
	arr, ok := s.BoolStatuses[id]
	if !ok {
		return 0
	}
	count := 0
	for _, v := range arr {
		if v {
			count++
		}
	}
	return count
}

func (s *Score) AnyEnumStatus(id string, atLeastIndex int) bool {
	s.ensureInit()
	arr, ok := s.EnumStatuses[id]
	if !ok {
		return false
	}
	for _, v := range arr {
		if v >= atLeastIndex {
			return true
		}
	}
	return false
}

func (s *Score) CountEnumStatus(id string, atLeastIndex int) int {
	s.ensureInit()
	arr, ok := s.EnumStatuses[id]
	if !ok {
		return 0
	}
	count := 0
	for _, v := range arr {
		if v >= atLeastIndex {
			count++
		}
	}
	return count
}

func (s *Score) Clone() *Score {
	if s == nil {
		return nil
	}
	s.ensureInit()
	c := &Score{
		Fouls:     append([]Foul(nil), s.Fouls...),
		PlayoffDq: s.PlayoffDq,
		Hub:       s.Hub,
		CustomScoreData: CustomScoreData{
			Counts:       make(map[string]int, len(s.Counts)),
			BoolStatuses: make(map[string][3]bool, len(s.BoolStatuses)),
			EnumStatuses: make(map[string][3]int, len(s.EnumStatuses)),
		},
	}
	for k, v := range s.Counts {
		c.Counts[k] = v
	}
	for k, v := range s.BoolStatuses {
		c.BoolStatuses[k] = v
	}
	for k, v := range s.EnumStatuses {
		c.EnumStatuses[k] = v
	}
	return c
}

// CopyInto overwrites dst with a deep copy of the score, reusing dst's existing backing storage
// where possible. The arena calls this once per 10 ms tick to keep a snapshot for change detection,
// so it must not allocate in the steady state: the maps and the Fouls slice are refilled in place
// rather than reallocated, which is what makes the custom build's map-based score affordable there.
func (s *Score) CopyInto(dst *Score) {
	s.ensureInit()
	dst.ensureInit()

	fouls := dst.Fouls[:0]
	counts, boolStatuses, enumStatuses := dst.Counts, dst.BoolStatuses, dst.EnumStatuses
	clear(counts)
	clear(boolStatuses)
	clear(enumStatuses)

	*dst = *s
	dst.Fouls = append(fouls, s.Fouls...)
	dst.Counts = counts
	dst.BoolStatuses = boolStatuses
	dst.EnumStatuses = enumStatuses

	for k, v := range s.Counts {
		counts[k] = v
	}
	for k, v := range s.BoolStatuses {
		boolStatuses[k] = v
	}
	for k, v := range s.EnumStatuses {
		enumStatuses[k] = v
	}
}

func (s *Score) Equals(other *Score) bool {
	if s == nil || other == nil {
		return s == other
	}
	if s.PlayoffDq != other.PlayoffDq || len(s.Fouls) != len(other.Fouls) {
		return false
	}
	for i, f := range s.Fouls {
		if f != other.Fouls[i] {
			return false
		}
	}
	s.ensureInit()
	other.ensureInit()

	for k, v := range s.Counts {
		if other.Counts[k] != v {
			return false
		}
	}
	for k, v := range other.Counts {
		if s.Counts[k] != v {
			return false
		}
	}

	for k, v := range s.BoolStatuses {
		if other.BoolStatuses[k] != v {
			return false
		}
	}
	for k, v := range other.BoolStatuses {
		if s.BoolStatuses[k] != v {
			return false
		}
	}

	for k, v := range s.EnumStatuses {
		if other.EnumStatuses[k] != v {
			return false
		}
	}
	for k, v := range other.EnumStatuses {
		if s.EnumStatuses[k] != v {
			return false
		}
	}

	return true
}
