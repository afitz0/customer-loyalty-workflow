import asyncio
import logging
from temporalio.client import Client
from shared import Customer, STATUS_LEVELS
from loyalty_workflow import CustomerLoyaltyWorkflow, TASK_QUEUE


async def main():
    logging.basicConfig(level=logging.INFO)

    client = await Client.connect("localhost:7233")

    customer = Customer(name="Customer", id="123", tier=STATUS_LEVELS[-1])

    await client.start_workflow(
        CustomerLoyaltyWorkflow.run,
        customer,
        id=CustomerLoyaltyWorkflow.workflow_id(customer.id),
        task_queue=TASK_QUEUE,
    )


if __name__ == "__main__":
    asyncio.run(main())
