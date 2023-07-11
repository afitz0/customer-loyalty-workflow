import asyncio

from temporalio.client import Client
from temporalio.worker import Worker

from loyalty_workflow import *
from shared import TASK_QUEUE


async def main():
    # Uncomment the line below to see logging
    logging.basicConfig(level=logging.INFO)

    # Start client
    client = await Client.connect("localhost:7233")

    worker = Worker(
        client,
        task_queue=TASK_QUEUE,
        workflows=[CustomerLoyaltyWorkflow],
        activities=[send_email],
    )
    logging.info("Starting worker.")
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
