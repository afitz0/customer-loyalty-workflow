import logging
from datetime import timedelta

from temporalio import activity, workflow
from temporalio.common import WorkflowIDReusePolicy
from temporalio.exceptions import WorkflowAlreadyStartedError
from temporalio.workflow import ParentClosePolicy
from temporalio.client import WorkflowHandle

import email_strings
from shared import *


# Basic activity that logs and does string concatenation
@activity.defn(name="SendEmail")
async def send_email(body: str):
    activity.logger.info("Sending email with contents %s" % body)


# Basic workflow that logs and invokes an activity
@workflow.defn
class CustomerLoyaltyWorkflow:
    def __init__(self):
        self.customer = None

    @workflow.run
    async def run(self, customer: Customer) -> str:
        self.customer = customer
        workflow.logger.info("Running workflow with parameter %s" % customer)
        info = workflow.info()

        if not info.continued_run_id:
            await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_WELCOME.format(self.customer.status.name),
                start_to_close_timeout=timedelta(seconds=5),
            )

        await workflow.wait_condition(
            lambda: not self.customer.account_active or info.get_current_history_length() > EVENT_HISTORY_THRESHOLD
        )

        if self.customer.account_active:
            logging.info(
                "Account %s still active, but event history threshold reached; continuing-as-new." % self.customer.customerId)
            workflow.continue_as_new(self.customer)

        return "Loyalty workflow completed. Customer: %s" % self.customer.customerId

    @workflow.signal(name=SIGNAL_CANCEL_ACCOUNT)
    async def cancel_account(self) -> None:
        self.customer.account_active = False
        await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_CANCEL_ACCOUNT,
                start_to_close_timeout=timedelta(seconds=5),
            )

    @workflow.signal(name=SIGNAL_ADD_POINTS)
    async def add_points(self, points_to_add: int) -> None:
        self.customer.points += points_to_add

        status_change = self.customer.status.update(self.customer.points)
        if status_change > 0:
            await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_PROMOTED.format(self.customer.status.name),
                start_to_close_timeout=timedelta(seconds=5)
            )
        elif status_change < 0:
            await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_DEMOTED.format(self.customer.status.name),
                start_to_close_timeout=timedelta(seconds=5)
            )

    @workflow.signal(name=SIGNAL_INVITE_GUEST)
    async def invite_guest(self, guest_id: str) -> None:
        if len(self.customer.guests) >= self.customer.status.guests_allowed:
            await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_INSUFFICIENT_POINTS,
                start_to_close_timeout=timedelta(seconds=5),
            )
            return

        self.customer.guests.add(guest_id)

        guest = Customer(customerId=guest_id)
        child_workflow_id = CUSTOMER_WORKFLOW_ID_FORMAT.format(guest_id)
        try:
            child_handle = await workflow.start_child_workflow(
                CustomerLoyaltyWorkflow.run,
                guest,
                id=child_workflow_id,
                parent_close_policy=ParentClosePolicy.ABANDON,
                id_reuse_policy=WorkflowIDReusePolicy.REJECT_DUPLICATE,
            )
        except WorkflowAlreadyStartedError:
            logging.info("Child workflow already started")
            child_handle: WorkflowHandle = workflow.get_external_workflow_handle(child_workflow_id)

        child_info = await child_handle.describe()
        isclosed = child_info.close_time is not None

        if isclosed:
            await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_GUEST_CANCELED,
                start_to_close_timeout=timedelta(seconds=5),
            )
        else:
            await workflow.execute_activity(
                send_email,
                email_strings.EMAIL_GUEST_INVITED,
                start_to_close_timeout=timedelta(seconds=5),
            )
            await child_handle.signal(SIGNAL_ENSURE_MINIMUM_STATUS, self.customer.status.previous())

    @workflow.signal(name=SIGNAL_ENSURE_MINIMUM_STATUS)
    async def ensure_minimum_status(self, min_status: StatusTier):
        self.customer.status.ensure_at_least(min_status)

    @workflow.query(name=QUERY_GET_STATUS)
    def get_status(self) -> GetStatusResponse:
        return GetStatusResponse(
            points=self.customer.points,
            account_active=self.customer.account_active,
            status_level=self.customer.status_level,
            tier=self.customer.status.tier
        )

    @workflow.query(name=QUERY_GET_GUESTS)
    def get_guests(self) -> list[str]:
        return list(map(lambda c: c.customerId, self.customer.guests))
