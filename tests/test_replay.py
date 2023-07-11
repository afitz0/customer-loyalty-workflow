import os
import json
import pytest
from temporalio.client import WorkflowHistory
from temporalio.worker import Replayer

from loyalty_workflow import CustomerLoyaltyWorkflow


@pytest.mark.asyncio
async def test_simple_replay():
    with open(os.path.dirname(os.path.realpath(__file__)) + '/simple_replay.json') as f:
        history_json = json.load(f)

    replayer = Replayer(workflows=[CustomerLoyaltyWorkflow])
    await replayer.replay_workflow(WorkflowHistory.from_json("customer-123", history_json))

