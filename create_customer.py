import logging
from temporalio.client import Client
from shared import *
from loyalty_workflow import CustomerLoyaltyWorkflow


def main():
    logging.basicConfig(level=logging.INFO)

    client = await Client.connect("localhost:7233")

    customer = Customer(name="Customer", id="123")

    client.execute_workflow(
        CustomerLoyaltyWorkflow.run,
        customer,
        id=CUSTOMER_WORKFLOW_ID_FORMAT.format(customer.id),
        task_queue=TASK_QUEUE,
    )


if __name__ == "__main__":
    main()
