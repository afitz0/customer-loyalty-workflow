import logging
import uuid

import pytest
from temporalio import activity
from temporalio.client import WorkflowExecutionStatus
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import Worker

from loyalty_workflow import CustomerLoyaltyWorkflow
from shared import Customer, STATUS_LEVELS, GetStatusResponse, LoyaltyWorkflowInput


class DefaultActivityMocks:
    @activity.defn(name="send_email")
    async def send_email_mock(self, body: str) -> None:
        return

    @activity.defn(name="start_guest_workflow")
    async def start_guest_workflow_mock(self, guest: Customer) -> bool:
        return True


@pytest.mark.asyncio
async def test_execute_workflow() -> None:
    task_queue_name = str(uuid.uuid4())
    logging.basicConfig(level=logging.INFO)

    acts = DefaultActivityMocks()
    async with await WorkflowEnvironment.start_time_skipping() as env:
        async with Worker(
                env.client,
                task_queue=task_queue_name,
                workflows=[CustomerLoyaltyWorkflow],
                activities=[acts.send_email_mock, acts.start_guest_workflow_mock],
        ):
            handle = await env.client.start_workflow(
                CustomerLoyaltyWorkflow.run,
                arg=LoyaltyWorkflowInput(customer=Customer(id="123")),
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            await env.sleep(1)
            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)
            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status

            assert "workflow completed" in await handle.result()


@pytest.mark.asyncio
async def test_add_points_single_promo() -> None:
    task_queue_name = str(uuid.uuid4())
    logging.basicConfig(level=logging.INFO)

    acts = DefaultActivityMocks()
    async with await WorkflowEnvironment.start_time_skipping() as env:
        async with Worker(
                env.client,
                task_queue=task_queue_name,
                workflows=[CustomerLoyaltyWorkflow],
                activities=[acts.send_email_mock, acts.start_guest_workflow_mock],
        ):
            handle = await env.client.start_workflow(
                CustomerLoyaltyWorkflow.run,
                arg=LoyaltyWorkflowInput(customer=Customer(id="123")),
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            await handle.signal(CustomerLoyaltyWorkflow.add_points, STATUS_LEVELS[1].minimum_points)
            await env.sleep(1)

            status: GetStatusResponse = await handle.query(CustomerLoyaltyWorkflow.get_status)
            assert STATUS_LEVELS[1].name == status.tier.name

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)

            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status
            assert "workflow completed" in await handle.result()


@pytest.mark.asyncio
async def test_add_points_multi_promo() -> None:
    task_queue_name = str(uuid.uuid4())
    logging.basicConfig(level=logging.INFO)

    acts = DefaultActivityMocks()
    async with await WorkflowEnvironment.start_time_skipping() as env:
        async with Worker(
                env.client,
                task_queue=task_queue_name,
                workflows=[CustomerLoyaltyWorkflow],
                activities=[acts.send_email_mock, acts.start_guest_workflow_mock],
        ):
            handle = await env.client.start_workflow(
                CustomerLoyaltyWorkflow.run,
                arg=LoyaltyWorkflowInput(customer=Customer(id="123")),
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            await handle.signal(CustomerLoyaltyWorkflow.add_points, STATUS_LEVELS[2].minimum_points)
            await env.sleep(1)

            status: GetStatusResponse = await handle.query(CustomerLoyaltyWorkflow.get_status)
            assert STATUS_LEVELS[2].name == status.tier.name

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)

            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status
            assert "workflow completed" in await handle.result()


@pytest.mark.asyncio
async def test_invite_guest() -> None:
    task_queue_name = str(uuid.uuid4())
    logging.basicConfig(level=logging.INFO)

    acts = DefaultActivityMocks()
    async with await WorkflowEnvironment.start_time_skipping() as env:
        async with Worker(
                env.client,
                task_queue=task_queue_name,
                workflows=[CustomerLoyaltyWorkflow],
                activities=[acts.send_email_mock, acts.start_guest_workflow_mock],
        ):
            handle = await env.client.start_workflow(
                CustomerLoyaltyWorkflow.run,
                arg=LoyaltyWorkflowInput(customer=Customer(id="123", tier=STATUS_LEVELS[-1])),
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            # give time for workflow to start before attempting to invite
            await env.sleep(1)
            await handle.signal(CustomerLoyaltyWorkflow.invite_guest, "guest")

            await env.sleep(1)
            guests = await handle.query(CustomerLoyaltyWorkflow.get_guests)
            assert "guest" in guests

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)

            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status
            assert "workflow completed" in await handle.result()


@pytest.mark.asyncio
async def test_invite_canceled_guest() -> None:
    task_queue_name = str(uuid.uuid4())
    logging.basicConfig(level=logging.INFO)

    cancel_email_call_count: int = 0

    @activity.defn(name="send_email")
    async def send_email_mocked(body: str) -> None:
        nonlocal cancel_email_call_count
        if "guest has already canceled" in body:
            cancel_email_call_count += 1

    @activity.defn(name="start_guest_workflow")
    async def start_guest_workflow_mocked(guest: Customer) -> bool:
        assert guest.tier == STATUS_LEVELS[-2]
        return False

    async with await WorkflowEnvironment.start_time_skipping() as env:
        async with Worker(
                env.client,
                task_queue=task_queue_name,
                workflows=[CustomerLoyaltyWorkflow],
                activities=[send_email_mocked, start_guest_workflow_mocked],
        ):
            handle = await env.client.start_workflow(
                CustomerLoyaltyWorkflow.run,
                arg=LoyaltyWorkflowInput(customer=Customer(id="123", tier=STATUS_LEVELS[-1])),
                id=str(uuid.uuid4()),
                task_queue=task_queue_name
            )

            # give time for workflow to start before attempting to invite
            await env.sleep(1)

            await handle.signal(CustomerLoyaltyWorkflow.invite_guest, "guest")

            await env.sleep(1)
            guests = await handle.query(CustomerLoyaltyWorkflow.get_guests)
            assert "guest" in guests

            await handle.signal(CustomerLoyaltyWorkflow.cancel_account)
            await env.sleep(1)

            assert WorkflowExecutionStatus.COMPLETED == (await handle.describe()).status
            assert "workflow completed" in await handle.result()
            assert cancel_email_call_count == 1, "Expected 'guest has canceled' to be called exactly once."
