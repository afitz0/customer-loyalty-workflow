from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class Customer:
    id: str = ""
    name: str = ""
    points: int = 0
    guests: list[str] = field(default_factory=list)
    account_active: bool = True
    tier: StatusTier = field(default_factory=lambda: StatusTier())


@dataclass(frozen=True)
class StatusTier:
    name: str = "Member"
    minimum_points: int = 0
    guests_allowed: int = 0
    level: int = 0

    @staticmethod
    def status_for_points(points: int) -> StatusTier:
        for (i, level) in enumerate(STATUS_LEVELS):
            if i > 0 and points < level.minimum_points:
                return STATUS_LEVELS[i - 1]
        return STATUS_LEVELS[-1]

    @staticmethod
    def previous(tier: StatusTier) -> StatusTier:
        if tier.level > 0:
            return STATUS_LEVELS[tier.level - 1]
        return STATUS_LEVELS[0]


STATUS_LEVELS: list[StatusTier] = [
    StatusTier(),
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
