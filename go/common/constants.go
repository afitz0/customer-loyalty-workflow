package common

const (
	TaskQueue                = "CustomerLoyaltyTaskQueue"
	CustomerWorkflowIDFormat = "customer-%v"
	EventsThreshold          = 10000

	SignalCancelAccount       = "cancelAccount"
	SignalAddPoints           = "addLoyaltyPoints"
	SignalInviteGuest         = "inviteGuest"
	SignalEnsureMinimumStatus = "ensureMinimumStatus"
	QueryGetStatus            = "getStatus"
	QueryGetGuests            = "getGuests"
)

const (
	EmailWelcome            = "Welcome to our loyalty program! You're starting out at '%v' status."
	EmailGuestCanceled      = "Sorry, your guest has already canceled their account."
	EmailGuestInvited       = "Congratulations! Your guest has been invited!"
	EmailInsufficientPoints = "Sorry, you need to earn more points to invite more guests!"
	EmailPromoted           = "Congratulations! You've been promoted to '%v' status!"
	EmailDemoted            = "Unfortunately, you've lost enough points to bump you down to '%v' status. 😞"
	EmailCancelAccount      = "Sorry to see you go!"
)
