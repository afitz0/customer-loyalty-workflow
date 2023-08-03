import asyncio
import logging

from temporalio.client import Client
from temporalio.worker import Worker

from loyalty_workflow import CustomerLoyaltyWorkflow, TASK_QUEUE
from activities import LoyaltyActivities


async def main() -> None:
    logging.basicConfig(level=logging.INFO)

    # Start client
    client: Client = await Client.connect("localhost:7233")

    activities = LoyaltyActivities(client)

    worker = Worker(
        client,
        task_queue=TASK_QUEUE,
        workflows=[CustomerLoyaltyWorkflow],
        activities=[activities.send_email, activities.start_guest_workflow],
    )
    logging.info("Starting worker.")
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
