package status

var Levels = []Tier{
	{
		Name:          "Member",
		MinimumPoints: 0,
		GuestsAllowed: 0,
	},
	{
		Name:          "Bronze",
		MinimumPoints: 500,
		GuestsAllowed: 1,
	},
	{
		Name:          "Silver",
		MinimumPoints: 1000,
		GuestsAllowed: 2,
	},
	{
		Name:          "Gold",
		MinimumPoints: 2000,
		GuestsAllowed: 5,
	},
	{
		Name:          "Platinum",
		MinimumPoints: 5000,
		GuestsAllowed: 10,
	},
}

type (
	Status interface {
		// Update modifies the current status level based on a given points value, comparing to the minimum required points for
		// each status level. The change from previous to new is returned. For example, if the customer now has enough points to
		// warrant the next level, Update will return 1, but if the new point value doesn't cross the next level's minimum,
		// Update will return 0.
		Update(points int) (change int)

		// Name returns the string representation of the current status level.
		Name() string

		// EnsureMinimum sets the current status level to match the level of the given tier, returning true if it resulted in a
		// change to a higher level.
		EnsureMinimum(tier Tier) bool

		// PreviousTier returns the Tier for one level lower than the current, minimum lowest tier.
		PreviousTier() Tier

		// NumGuestsAllowed returns the number of guests allowed at the current level.
		NumGuestsAllowed() int

		// Tier returns the current Tier for this level.
		Tier() Tier
	}

	Level struct {
		level int
	}

	Tier struct {
		Name          string
		MinimumPoints int
		GuestsAllowed int
	}
)

func NewStatus(level int) Status {
	// Guard for known levels.
	if level >= len(Levels) {
		level = len(Levels) - 1
	}
	if level < 0 {
		level = 0
	}
	return &Level{level}
}

func (s *Level) Update(points int) (change int) {
	newLevel := 0
	for i, l := range Levels {
		if points >= l.MinimumPoints {
			newLevel = i
		}
	}

	change = newLevel - s.level
	s.level = newLevel
	return change
}

func (s *Level) Name() string {
	return Levels[s.level].Name
}

func (s *Level) EnsureMinimum(tier Tier) bool {
	var minLevel int
	for i, t := range Levels {
		if t.Name == tier.Name {
			minLevel = i
		}
	}

	if s.level < minLevel {
		s.level = minLevel
		return true
	}

	return false
}

func (s *Level) PreviousTier() Tier {
	if s.level == 0 {
		return Levels[0]
	}
	return Levels[s.level-1]
}

func (s *Level) NumGuestsAllowed() int {
	return Levels[s.level].GuestsAllowed
}

func (s *Level) Tier() Tier {
	return Levels[s.level]
}
