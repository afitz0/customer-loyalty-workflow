import logging
import uuid

import pytest

# from temporalio import activity
from temporalio.client import WorkflowExecutionStatus
from temporalio.worker import Worker
from temporalio.testing import WorkflowEnvironment


from loyalty_workflow import (
    CustomerLoyaltyWorkflow,
)
from activities import send_email
from shared import Customer


@pytest.mark.asyncio
async def test_execute_workflow():
    task_queue_name = str(uuid.uuid4())
    logging.basicConfig(level=logging.INFO)

    async with await WorkflowEnvironment.start_time_skipping() as env:
        async with Worker(
            env.client,
            task_queue=task_queue_name,
            workflows=[CustomerLoyaltyWorkflow],
            activities=[send_email],
        ):
            handle = await env.client.start_workflow(
                CustomerLoyaltyWorkflow.run,
                Customer(customerId="123"),
                id=str(uuid.uuid4()), task_queue=task_queue_name
            )

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)
            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status

            assert "Loyalty workflow completed. Customer: 123" == await handle.result()
