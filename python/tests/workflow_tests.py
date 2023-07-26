import logging
import uuid

import pytest
# from temporalio import activity
from temporalio.client import WorkflowExecutionStatus
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import Worker

from activities import send_email
from loyalty_workflow import CustomerLoyaltyWorkflow
from shared import Customer, STATUS_LEVELS, GetStatusResponse


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
                args=[Customer(id="123"), True],
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)
            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status

            assert "workflow completed" in await handle.result()


@pytest.mark.asyncio
async def test_add_points_single_promo():
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
                args=[Customer(id="123"), True],
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            await handle.signal(CustomerLoyaltyWorkflow.add_points, STATUS_LEVELS[1].minimum_points)
            status: GetStatusResponse = await handle.query(CustomerLoyaltyWorkflow.get_status)
            assert STATUS_LEVELS[1].name == status.tier.name

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)
            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status

            assert "workflow completed" in await handle.result()


@pytest.mark.asyncio
async def test_add_points_multi_promo():
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
                args=[Customer(id="123"), True],
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            await handle.signal(CustomerLoyaltyWorkflow.add_points, STATUS_LEVELS[2].minimum_points)
            status: GetStatusResponse = await handle.query(CustomerLoyaltyWorkflow.get_status)
            assert STATUS_LEVELS[2].name == status.tier.name

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)
            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status

            assert "workflow completed" in await handle.result()
