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
