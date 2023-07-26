from temporalio import activity, common
from temporalio.client import Client
from temporalio.exceptions import WorkflowAlreadyStartedError

from shared import Customer



@activity.defn
async def send_email(body: str) -> None:
    activity.logger.info("Sending email with contents %s" % body)


@activity.defn
async def start_guest_workflow(guest: Customer) -> bool:
    from loyalty_workflow import (
        CustomerLoyaltyWorkflow,
        TASK_QUEUE,
        SIGNAL_ENSURE_MINIMUM_STATUS
    )

    activity.logger.info("Starting guest workflow with ID %s" % guest.id)

    # TODO how do I share this from the worker?
    client = await Client.connect("localhost:7233")

    try:
        await client.start_workflow(
            CustomerLoyaltyWorkflow.run,
            id=CustomerLoyaltyWorkflow.workflow_id(guest.id),
            task_queue=TASK_QUEUE,
            start_signal=SIGNAL_ENSURE_MINIMUM_STATUS,
            start_signal_args=[guest.tier],
            id_reuse_policy=common.WorkflowIDReusePolicy.REJECT_DUPLICATE,
        )
    except WorkflowAlreadyStartedError:
        return False

    return True
