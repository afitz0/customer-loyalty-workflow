package loyalty

import "github.com/afitz0/customer-loyalty-workflow/status"

type (
	CustomerInfo struct {
		CustomerId    string
		LoyaltyPoints int
		StatusLevel   int
		Status        status.Status
		Name          string
		Guests        map[string]struct{}
		AccountActive bool
	}

	GetStatusResponse struct {
		Tier          status.Tier
		Points        int
		AccountActive bool
	}
)
