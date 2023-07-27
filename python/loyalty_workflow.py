import logging
from datetime import timedelta
from enum import StrEnum
from typing import List, Any

from temporalio import workflow

from activities import LoyaltyActivities

with workflow.unsafe.imports_passed_through():
    from shared import (
        Customer,
        GetStatusResponse,
        StatusTier
    )

TASK_QUEUE = "CustomerLoyaltyTaskQueue"
EVENT_HISTORY_THRESHOLD = 10_000


class Signals(StrEnum):
    CANCEL_ACCOUNT = "cancelAccount"
    ADD_POINTS = "addLoyaltyPoints"
    INVITE_GUEST = "inviteGuest"
    ENSURE_MINIMUM_STATUS = "ensureMinimumStatus"


class Queries(StrEnum):
    GET_STATUS = "getStatus"
    GET_GUESTS = "getGuests"


@workflow.defn
class CustomerLoyaltyWorkflow:
    def __init__(self) -> None:
        self.customer: Customer = Customer()
        self._signal_queue: List[tuple[str, Any]] = []

    @workflow.run
    async def run(self, customer: Customer, is_new: bool = True) -> str:
        logging.basicConfig(level=logging.INFO)

        self.customer = customer
        self.validate_inputs()

        workflow.logger.info("Running workflow with parameter %s" % self.customer)
        info: workflow.Info = workflow.info()

        if is_new:
            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Welcome to our loyalty program! You're starting out at '{}' status.".format(self.customer.tier.name),
                start_to_close_timeout=timedelta(seconds=5),
            )

        while True:
            await workflow.wait_condition(
                lambda: len(self._signal_queue) > 0 or not self.customer.account_active
            )

            while len(self._signal_queue) > 0:
                signal = self._signal_queue.pop(0)
                await self.process_signal(signal_name=signal[0], arg=signal[1])

            if not self.customer.account_active or info.get_current_history_length() > EVENT_HISTORY_THRESHOLD:
                break

        if self.customer.account_active:
            logging.info(
                "Account %s still active, but event history threshold reached; continuing-as-new."
                % self.customer.id)
            workflow.continue_as_new(args=[self.customer, False])

        return "Loyalty workflow completed. Customer: %s" % self.customer.id

    async def process_signal(self, signal_name: str, arg: Any) -> None:
        match signal_name:
            case Signals.ADD_POINTS:
                await self._process_add_points(arg)
            case Signals.INVITE_GUEST:
                await self._process_invite_guest(arg)
            case Signals.ENSURE_MINIMUM_STATUS:
                await self._process_ensure_minimum_status(arg)
            case Signals.CANCEL_ACCOUNT:
                await self._process_cancel_account()

    @workflow.signal(name=Signals.CANCEL_ACCOUNT)
    async def cancel_account(self) -> None:
        self._signal_queue.append((Signals.CANCEL_ACCOUNT, None))

    async def _process_cancel_account(self) -> None:
        self.customer.account_active = False
        await workflow.execute_activity(
            LoyaltyActivities.send_email,
            "Sorry to see you go!",
            start_to_close_timeout=timedelta(seconds=5),
        )

    @workflow.signal(name=Signals.ADD_POINTS)
    async def add_points(self, points_to_add: int) -> None:
        self._signal_queue.append((Signals.ADD_POINTS, points_to_add))

    async def _process_add_points(self, points_to_add: int) -> None:
        self.customer.points += points_to_add

        new_tier = StatusTier.status_for_points(self.customer.points)
        status_change = new_tier.level - self.customer.tier.level
        self.customer.tier = new_tier

        if status_change > 0:
            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Congratulations! You've been promoted to '{}' status!".format(self.customer.tier.name),
                start_to_close_timeout=timedelta(seconds=5)
            )
        elif status_change < 0:
            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Unfortunately, you've lost enough points to bump you down to '{}' status. 😞"
                .format(self.customer.tier.name),
                start_to_close_timeout=timedelta(seconds=5)
            )

    @workflow.signal(name=Signals.INVITE_GUEST)
    async def invite_guest(self, guest_id: str) -> None:
        self._signal_queue.append((Signals.INVITE_GUEST, guest_id))

    async def _process_invite_guest(self, guest_id: str) -> None:
        if len(self.customer.guests) >= self.customer.tier.guests_allowed:
            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Sorry, you need to earn more points to invite more guests!",
                start_to_close_timeout=timedelta(seconds=5),
            )
            return

        if guest_id not in self.customer.guests:
            self.customer.guests.append(guest_id)

        previous = StatusTier.previous(self.customer.tier)
        guest = Customer(
            id=guest_id,
            tier=previous,
        )

        started: bool = await workflow.execute_activity(
            LoyaltyActivities.start_guest_workflow,
            guest,
            start_to_close_timeout=timedelta(seconds=5),
        )

        if started:
            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Congratulations! Your guest has been invited!",
                start_to_close_timeout=timedelta(seconds=5),
            )
        else:
            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Sorry, your guest has already canceled their account.",
                start_to_close_timeout=timedelta(seconds=5),
            )

    @workflow.signal(name=Signals.ENSURE_MINIMUM_STATUS)
    async def ensure_minimum_status(self, min_status: StatusTier) -> None:
        self._signal_queue.append((Signals.ENSURE_MINIMUM_STATUS, min_status))

    async def _process_ensure_minimum_status(self, min_status: StatusTier) -> None:
        if self.customer.tier.level < min_status.level:
            self.customer.tier = min_status
            self.customer.points = min_status.minimum_points

            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Congratulations! You've been promoted to '{}' status!".format(self.customer.tier.name),
                start_to_close_timeout=timedelta(seconds=5)
            )

    @workflow.query(name=Queries.GET_STATUS)
    def get_status(self) -> GetStatusResponse:
        return GetStatusResponse(
            points=self.customer.points,
            account_active=self.customer.account_active,
            status_level=self.customer.tier.level,
            tier=self.customer.tier
        )

    @workflow.query(name=Queries.GET_GUESTS)
    def get_guests(self) -> list[str]:
        return list(self.customer.guests)

    def validate_inputs(self) -> None:
        # Note: I found that without this, I got a "'dict' object has no attribute 'tier'" error thrown.
        if isinstance(self.customer, dict):
            self.customer = Customer(**self.customer)
        if isinstance(self.customer.tier, dict):
            self.customer.tier = StatusTier(**self.customer.tier)

    @staticmethod
    def workflow_id(customer_id: str) -> str:
        return "customer-{}".format(customer_id)
