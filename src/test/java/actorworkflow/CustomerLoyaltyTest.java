package actorworkflow;

import static org.junit.Assert.assertEquals;
import static org.mockito.Mockito.mock;

import io.temporal.api.enums.v1.WorkflowIdReusePolicy;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.testing.TestWorkflowRule;
import org.junit.Rule;
import org.junit.Test;

import java.util.ArrayList;

/**
 * Unit test for {@link CustomerLoyaltyWorkflow}.
 */
public class CustomerLoyaltyTest {

    @Rule
    public TestWorkflowRule testWorkflowRule =
            TestWorkflowRule.newBuilder()
                    .setWorkflowTypes(CustomerLoyaltyWorkflowImpl.class)
                    .setActivityImplementations(new CustomerLoyaltyActivitiesImpl())
                    .build();

    @Test
    public void testAddPoints() {
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
        WorkflowClient.start(workflow::customerLoyalty, new Customer("123"));

        StatusTier targetStatus = StatusTier.STATUS_TIERS.get(1);
        workflow.addLoyaltyPoints(targetStatus.minimumPoints());
        StatusTier customerStatus = workflow.getStatus();
        assertEquals(customerStatus, targetStatus);
    }


    @Test
    public void testAddGuest() {
        CustomerLoyaltyActivities activities = mock(CustomerLoyaltyActivities.class);
        testWorkflowRule.getWorker().registerActivitiesImplementations(activities);

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
        var customer = new Customer("host");
        customer = customer.withStatus(new StatusTier("", 0, 1000));
        WorkflowClient.start(workflow::customerLoyalty, customer);

        var guest = new Customer("guest");
        workflow.inviteGuest(guest);
    }
}
