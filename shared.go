package starter

const TaskQueue = "CustomerLoyaltyTaskQueue"
const CustomerWorkflowIdFormat = "customer-%v"
const EventsThreshold = 10000

const (
	SignalCancelAccount       = "cancelAccount"
	SignalAddPoints           = "addLoyaltyPoints"
	SignalInviteGuest         = "inviteGuest"
	SignalEnsureMinimumStatus = "ensureMinimumStatus"
	QueryGetStatus            = "getStatus"
	QueryGetGuests            = "getGuests"
)

type StatusTier struct {
	Name          string
	MinimumPoints int
	GuestsAllowed int
}

type GetStatusResponse struct {
	StatusLevel   int
	Tier          StatusTier
	Points        int
	AccountActive bool
}

var StatusTiers = []StatusTier{
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

type CustomerInfo struct {
	CustomerId    string
	LoyaltyPoints int
	StatusLevel   int
	Name          string
	Guests        map[string]struct{}
	AccountActive bool
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
