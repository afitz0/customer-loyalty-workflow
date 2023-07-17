package loyalty

type CustomerInfo struct {
	CustomerID    string
	LoyaltyPoints int
	StatusLevel   *StatusLevel
	Name          string
	Guests        []string
	AccountActive bool
}

type GetStatusResponse struct {
	StatusLevel   StatusLevel
	Points        int
	AccountActive bool
}

func (c *CustomerInfo) addGuest(guestID string) {
	// Add if not there
	for _, g := range c.Guests {
		if g == guestID {
			return
		}
	}
	c.Guests = append(c.Guests, guestID)
}

func (c *CustomerInfo) validate() {
	if c.StatusLevel == nil {
		c.StatusLevel = StatusLevelForPoints(c.LoyaltyPoints)
	}
}
