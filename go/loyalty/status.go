package loyalty

type StatusLevel struct {
	Ordinal       int
	Name          string
	MinimumPoints int
	GuestsAllowed int
}

func newStatusLevel(ord int, name string, minPoints int, guests int) *StatusLevel {
	return &StatusLevel{Ordinal: ord, Name: name, MinimumPoints: minPoints, GuestsAllowed: guests}
}

var StatusLevels = []*StatusLevel{
	newStatusLevel(0, "Member", 0, 0),
	newStatusLevel(1, "Bronze", 500, 1),
	newStatusLevel(2, "Silver", 1000, 2),
	newStatusLevel(3, "Gold", 2000, 5),
	newStatusLevel(4, "Platinum", 5000, 10),
}

func StatusLevelForPoints(points int) *StatusLevel {
	for i, level := range StatusLevels {
		if i > 0 && points < level.MinimumPoints {
			return StatusLevels[i-1]
		}
	}
	return StatusLevels[len(StatusLevels)-1]
}

// Previous returns nil if already at lowest, otherwise one StatusLevel lower than the current.
func (s *StatusLevel) Previous() *StatusLevel {
	if s.Ordinal > 0 {
		return StatusLevels[s.Ordinal-1]
	}
	return nil
}
