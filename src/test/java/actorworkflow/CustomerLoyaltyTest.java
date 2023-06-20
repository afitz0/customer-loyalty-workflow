package actorworkflow;

import static org.junit.Assert.assertEquals;

import io.temporal.api.enums.v1.WorkflowIdReusePolicy;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.testing.TestWorkflowRule;
import org.junit.Rule;
import org.junit.Test;

/** Unit test for {@link CustomerLoyaltyWorkflow}. */
public class CustomerLoyaltyTest {

    @Rule
    public TestWorkflowRule testWorkflowRule =
            TestWorkflowRule.newBuilder()
                    .setWorkflowTypes(CustomerLoyaltyWorkflowImpl.class)
                    .setActivityImplementations(new CustomerLoyaltyActivitiesImpl())
                    .build();

    @Test
    public void testSignal() {
        // Get a workflow stub using the same task queue the worker uses.
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setTaskQueue(testWorkflowRule.getTaskQueue())
                        .setWorkflowIdReusePolicy(
                                WorkflowIdReusePolicy.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE)
                        .build();
        CustomerLoyaltyWorkflow workflow =
                testWorkflowRule
                        .getWorkflowClient()
                        .newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        // Start workflow asynchronously to not use another thread to signal.
        WorkflowClient.start(workflow::customerLoyalty, new Customer("Test", "123"));

        StatusTier targetStatus = StatusTier.STATUS_TIERS.get(1);
        workflow.addLoyaltyPoints(targetStatus.minimumPoints());
        StatusTier customerStatus = workflow.getStatus();
        assertEquals(customerStatus, targetStatus);
    }
}
