package loyalty

import "github.com/afitz0/customer-loyalty-workflow/go/status"

type CustomerInfo struct {
	CustomerID    string
	LoyaltyPoints int
	StatusLevel   int
	Status        status.Status
	Name          string
	Guests        map[string]struct{}
	AccountActive bool
}

type GetStatusResponse struct {
	Tier          status.Tier
	Points        int
	AccountActive bool
}
