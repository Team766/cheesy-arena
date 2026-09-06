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

type CustomScoreData struct {
	Counts       map[string]int     `json:"counts"`
	BoolStatuses map[string][3]bool `json:"bool_statuses"`
	EnumStatuses map[string][3]int  `json:"enum_statuses"`
}

type Score struct {
	CustomScoreData
	Fouls     []Foul `json:"fouls"`
	PlayoffDq bool   `json:"playoff_dq"`
	Hub       Hub    `json:"hub"`
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

func (s *Score) AdjustCount(id string, phase Phase, delta int) bool {
	cfg := GetActiveConfig()
	if cfg != nil {
		valid := false
		for _, sc := range cfg.ScoringCounts {
			if sc.ID == id {
				for _, pp := range sc.Phases {
					if pp.Phase == phase.String() {
						valid = true
						break
					}
				}
				break
			}
		}
		if !valid {
			return false
		}
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

func (s *Score) SetBoolStatus(id string, robotIndex int, value bool) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	cfg := GetActiveConfig()
	if cfg != nil {
		valid := false
		for _, st := range cfg.Statuses {
			if st.ID == id && len(st.Values) == 0 {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
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

func (s *Score) SetEnumStatus(id string, robotIndex int, valueIndex int) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	cfg := GetActiveConfig()
	if cfg != nil {
		valid := false
		for _, st := range cfg.Statuses {
			if st.ID == id && len(st.Values) > 0 {
				if valueIndex >= 0 && valueIndex < len(st.Values) {
					valid = true
				}
				break
			}
		}
		if !valid {
			return false
		}
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
	cfg := GetActiveConfig()
	if cfg == nil {
		return false
	}
	valIdx := -1
	for _, st := range cfg.Statuses {
		if st.ID == id {
			for idx, v := range st.Values {
				if v.ID == valueId {
					valIdx = idx
					break
				}
			}
			break
		}
	}
	if valIdx < 0 {
		return false
	}
	return s.SetEnumStatus(id, robotIndex, valIdx)
}

func (s *Score) GetEnumStatus(id string, robotIndex int) int {
	if robotIndex < 0 || robotIndex >= 3 {
		return 0
	}
	s.ensureInit()
	return s.EnumStatuses[id][robotIndex]
}

func (s *Score) CycleEnumStatus(id string, robotIndex int) bool {
	if robotIndex < 0 || robotIndex >= 3 {
		return false
	}
	cfg := GetActiveConfig()
	numValues := 0
	if cfg != nil {
		for _, st := range cfg.Statuses {
			if st.ID == id {
				numValues = len(st.Values)
				break
			}
		}
	}
	if numValues == 0 {
		numValues = 3
	}

	curr := s.GetEnumStatus(id, robotIndex)
	next := (curr + 1) % numValues
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
