import json
from dataclasses import dataclass, field
from typing import Optional, Iterable, Any

from dataclasses_json import config

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
    id: str = field(metadata=config(field_name="customerId"))
    name: Optional[str] = ""
    points: Optional[int] = 0
    guests: Optional[set[str]] = field(default_factory=set)
    account_active: Optional[bool] = field(default_factory=lambda: True, metadata=config(field_name="accountActive"))
    status: Optional['Status'] = field(default_factory=lambda: Status(0))


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


class Status(Iterable[tuple[Any, Any]]):
    _level: int

    MEMBER = "Member"
    BRONZE = "Bronze"
    SILVER = "Silver"
    GOLD = "Gold"
    PLATINUM = "Platinum"

    LEVELS = [
        StatusTier(name=MEMBER, minimum_points=0, guests_allowed=0),
        StatusTier(name=BRONZE, minimum_points=500, guests_allowed=1),
        StatusTier(name=SILVER, minimum_points=1_000, guests_allowed=2),
        StatusTier(name=GOLD, minimum_points=2_000, guests_allowed=5),
        StatusTier(name=PLATINUM, minimum_points=5_000, guests_allowed=10),
    ]

    def __init__(self, level: Optional[int] = 0):
        """
        Init new customer status, starting at the given [optional] level..

        :param level:Numeric ranking for the level
        """
        self._level = max(level, 0)

    @property
    def level(self) -> int:
        return self._level

    @level.setter
    def level(self, value) -> None:
        value = max(min(len(Status.LEVELS) - 1, value), 0)
        self._level = value

    @property
    def name(self) -> str:
        return Status.LEVELS[self._level].name

    @property
    def minimum_points(self) -> int:
        return Status.LEVELS[self._level].minimum_points

    @property
    def guests_allowed(self) -> int:
        return Status.LEVELS[self._level].guests_allowed

    @property
    def tier(self) -> StatusTier:
        return Status.LEVELS[self._level]

    def update(self, points: int) -> int:
        """
        Given a points value, update this status to the appropriate level.

        :param points: the number of points a customer has.
        :return: the number of positions this status had to change with the given points. For example, if status moved
        from MEMBER to SILVER, return would be 2
        """
        new_level = 0
        for i, level in enumerate(Status.LEVELS):
            if points >= level.minimum_points:
                new_level = i

        diff = new_level - self._level
        self._level = new_level
        return diff

    def previous(self) -> 'Status':
        i = self._level
        return Status(i - 1)

    def ensure_at_least(self, status: StatusTier) -> None:
        min_level = Status.LEVELS.index(status)
        if self._level < min_level:
            self._level = min_level

    def __iter__(self):
        yield from {
            "level": self._level
        }.items()

    def __str__(self):
        return json.dumps(dict(self), ensure_ascii=False)

    def __repr__(self):
        return self.__str__()
