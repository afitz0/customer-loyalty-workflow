from __future__ import annotations
import json
from dataclasses import dataclass, field
from typing import Optional, Iterable, Any

from dataclasses_json import config


@dataclass
class Customer:
    customerId: str = field(default_factory=lambda: "", metadata=config(field_name="customerId"))
    name: str = ""
    points: int = 0
    guests: set[str] = field(default_factory=set)
    account_active: bool = field(default_factory=lambda: True, metadata=config(field_name="accountActive"))
    tier: StatusTier = field(default_factory=lambda: StatusTier())


@dataclass
class StatusTier:
    name: str = field(default_factory=lambda: STATUS_LEVELS[0].name)
    minimum_points: int = field(default_factory=lambda: STATUS_LEVELS[0].minimum_points)
    guests_allowed: int = field(default_factory=lambda: STATUS_LEVELS[0].guests_allowed)
    level: int = field(default_factory=lambda: STATUS_LEVELS[0].level)

    @staticmethod
    def status_for_points(points: int):
        for (i, level) in enumerate(STATUS_LEVELS):
            if i > 0 and points < level.minimum_points:
                return STATUS_LEVELS[i - 1]
        return STATUS_LEVELS[-1]


STATUS_LEVELS: list[StatusTier] = [
    StatusTier(name="Member", minimum_points=0, guests_allowed=0, level=0),
    StatusTier(name="Bronze", minimum_points=500, guests_allowed=1, level=1),
    StatusTier(name="Silver", minimum_points=1_000, guests_allowed=2, level=2),
    StatusTier(name="Gold", minimum_points=2_000, guests_allowed=5, level=3),
    StatusTier(name="Platinum", minimum_points=5_000, guests_allowed=10, level=4),
]


@dataclass
class GetStatusResponse:
    status_level: int
    tier: StatusTier
    points: int
    account_active: bool


class Status:
    LEVELS = [
        StatusTier(name="Member", minimum_points=0, guests_allowed=0, level=0),
        StatusTier(name="Bronze", minimum_points=500, guests_allowed=1, level=1),
        StatusTier(name="Silver", minimum_points=1_000, guests_allowed=2, level=2),
        StatusTier(name="Gold", minimum_points=2_000, guests_allowed=5, level=3),
        StatusTier(name="Platinum", minimum_points=5_000, guests_allowed=10, level=4),
    ]

    def __init__(self, level: int = 0):
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
        self._level = max(min(len(Status.LEVELS) - 1, value), 0)

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

    def previous(self) -> Status:
        i = self._level
        return Status(i - 1)

    def ensure_at_least(self, status: StatusTier) -> None:
        min_level = Status.LEVELS.index(status)
        if self._level < min_level:
            self._level = min_level
