import logging
from datetime import timedelta

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

# Signal and query names
SIGNAL_CANCEL_ACCOUNT = "cancelAccount"
SIGNAL_ADD_POINTS = "addLoyaltyPoints"
SIGNAL_INVITE_GUEST = "inviteGuest"
SIGNAL_ENSURE_MINIMUM_STATUS = "ensureMinimumStatus"
QUERY_GET_STATUS = "getStatus"
QUERY_GET_GUESTS = "getGuests"


@workflow.defn
class CustomerLoyaltyWorkflow:
    def __init__(self) -> None:
        self.customer: Customer = Customer()

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

        await workflow.wait_condition(
            lambda: not self.customer.account_active or info.get_current_history_length() > EVENT_HISTORY_THRESHOLD
        )

        if self.customer.account_active:
            logging.info(
                "Account %s still active, but event history threshold reached; continuing-as-new."
                % self.customer.id)
            workflow.continue_as_new(args=[self.customer, False])

        return "Loyalty workflow completed. Customer: %s" % self.customer.id

    @workflow.signal(name=SIGNAL_CANCEL_ACCOUNT)
    async def cancel_account(self) -> None:
        self.customer.account_active = False
        await workflow.execute_activity(
            LoyaltyActivities.send_email,
            "Sorry to see you go!",
            start_to_close_timeout=timedelta(seconds=5),
        )

    @workflow.signal(name=SIGNAL_ADD_POINTS)
    async def add_points(self, points_to_add: int) -> None:
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

    @workflow.signal(name=SIGNAL_INVITE_GUEST)
    async def invite_guest(self, guest_id: str) -> None:
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

    @workflow.signal(name=SIGNAL_ENSURE_MINIMUM_STATUS)
    async def ensure_minimum_status(self, min_status: StatusTier) -> None:
        if self.customer.tier.level < min_status.level:
            self.customer.tier = min_status
            self.customer.points = min_status.minimum_points

            await workflow.execute_activity(
                LoyaltyActivities.send_email,
                "Congratulations! You've been promoted to '{}' status!".format(self.customer.tier.name),
                start_to_close_timeout=timedelta(seconds=5)
            )

    @workflow.query(name=QUERY_GET_STATUS)
    def get_status(self) -> GetStatusResponse:
        return GetStatusResponse(
            points=self.customer.points,
            account_active=self.customer.account_active,
            status_level=self.customer.tier.level,
            tier=self.customer.tier
        )

    @workflow.query(name=QUERY_GET_GUESTS)
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
