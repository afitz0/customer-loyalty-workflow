from dataclasses import dataclass, field
from typing import Optional

TASK_QUEUE = "CustomerLoyaltyTaskQueue"
CUSTOMER_WORKFLOW_ID_FORMAT = "customer-{}"
EVENT_HISTORY_THRESHOLD = 10000

# Signal and query names
SIGNAL_CANCEL_ACCOUNT = "cancelAccount"
SIGNAL_ADD_POINTS = "addLoyaltyPoints"
SIGNAL_INVITE_GUEST = "inviteGuest"
SIGNAL_ENSURE_MINIMUM_STATUS = "ensureMinimumStatus"
QUERY_GET_STATUS = "getStatus"
QUERY_GET_GUESTS = "getGuests"


@dataclass
class Customer:
    id: str
    name: Optional[str] = ""
    points: Optional[int] = 0
    guests: Optional[list['Customer']] = field(default_factory=list)
    account_active: Optional[bool] = True


@dataclass
class StatusTier:
    name: str
    minimum_points: int
    guests_allowed: int


@dataclass
class GetStatusResponse:
    status_level: int
    tier: StatusTier
    points: int
    account_active: bool
