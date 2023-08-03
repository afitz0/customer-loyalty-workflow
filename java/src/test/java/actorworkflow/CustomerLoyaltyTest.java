package actorworkflow;

import io.temporal.api.enums.v1.WorkflowIdReusePolicy;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowExecutionAlreadyStarted;
import io.temporal.client.WorkflowOptions;
import io.temporal.client.WorkflowStub;
import io.temporal.testing.TestWorkflowRule;
import org.junit.Rule;
import org.junit.Test;

import java.time.Duration;
import java.util.ArrayList;

import static org.junit.Assert.assertEquals;
import static org.mockito.Mockito.*;

/**
 * Unit test for {@link CustomerLoyaltyWorkflow}.
 */
public class CustomerLoyaltyTest {

    @Rule
    public final TestWorkflowRule testWorkflowRule =
            TestWorkflowRule.newBuilder()
                    .setWorkflowTypes(CustomerLoyaltyWorkflowImpl.class)
                    .setDoNotStart(true)
                    .build();

    @Test
    public void testAddPoints() {
        testWorkflowRule.getWorker().registerActivitiesImplementations(
                new CustomerLoyaltyActivitiesImpl(testWorkflowRule.getWorkflowClient()));
        testWorkflowRule.getTestEnvironment().start();

        // Get a workflow stub using the same task queue the worker uses.
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setTaskQueue(testWorkflowRule.getTaskQueue())
                        .build();
        CustomerLoyaltyWorkflow workflow =
                testWorkflowRule
                        .getWorkflowClient()
                        .newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        // Start workflow asynchronously to not use another thread to signal.
        WorkflowClient.start(workflow::customerLoyalty, new Customer("123"));

        StatusTier targetStatus = StatusTier.STATUS_TIERS.get(1);

        int order = 0;
        testWorkflowRule.getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(order++),
                        () -> workflow.addLoyaltyPoints(targetStatus.minimumPoints()));

        testWorkflowRule.getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(order++), () -> {
                            StatusTier customerStatus = workflow.getStatus();
                            assertEquals(customerStatus, targetStatus);
                        }
                );

        testWorkflowRule.getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(order), workflow::cancelAccount);

        WorkflowStub.fromTyped(workflow).getResult(String.class);
        testWorkflowRule.getTestEnvironment().shutdown();
    }

    @Test
    public void testAddGuest() {
        CustomerLoyaltyActivities activities = mock(CustomerLoyaltyActivities.class);
        testWorkflowRule.getWorker().registerActivitiesImplementations(activities);
        testWorkflowRule.getTestEnvironment().start();

        var customer = new Customer("host", "", 0, StatusTier.STATUS_TIERS.get(4), new ArrayList<>());

        // Get a workflow stub using the same task queue the worker uses.
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setTaskQueue(testWorkflowRule.getTaskQueue())
                        .setWorkflowIdReusePolicy(
                                WorkflowIdReusePolicy.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY)
                        .setWorkflowId(CustomerLoyaltyWorkflow.workflowIdForCustomer(customer))
                        .build();
        CustomerLoyaltyWorkflow workflow =
                testWorkflowRule
                        .getWorkflowClient()
                        .newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        // Start workflow asynchronously to not use another thread to signal.
        WorkflowClient.start(workflow::customerLoyalty, customer);

        var guest = new Customer("guest");
        workflow.inviteGuest(guest);

        CustomerLoyaltyWorkflow child = testWorkflowRule
                .getWorkflowClient()
                .newWorkflowStub(CustomerLoyaltyWorkflow.class,
                        WorkflowOptions.newBuilder()
                                .setTaskQueue(testWorkflowRule.getTaskQueue())
                                .setWorkflowId(CustomerLoyaltyWorkflow.workflowIdForCustomer(guest))
                                .build());

        testWorkflowRule
                .getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(1), () -> {
                    // "start" the workflow, to make sure we have the current execution, but expect it to throw
                    try {
                        WorkflowClient.start(child::customerLoyalty, guest);
                    } catch (WorkflowExecutionAlreadyStarted ignored) {
                    }
                });

        testWorkflowRule
                .getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(2),
                        () -> assertEquals(child.getStatus(), StatusTier.STATUS_TIERS.get(3)));

        testWorkflowRule
                .getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(3), () -> {
                    workflow.cancelAccount();
                    child.cancelAccount();
                });

        WorkflowStub.fromTyped(workflow).getResult(String.class);
        WorkflowStub.fromTyped(child).getResult(String.class);
        testWorkflowRule.getTestEnvironment().shutdown();
    }

    @Test
    public void testAddGuestTwice() {
        CustomerLoyaltyActivities activities = mock(CustomerLoyaltyActivitiesImpl.class);
        when(activities.startGuestWorkflow(any(Customer.class), anyString()))
                .thenReturn(true)
                .thenReturn(false);

        testWorkflowRule.getWorker().registerActivitiesImplementations(activities);
        testWorkflowRule.getTestEnvironment().start();

        var customer = new Customer(
                "host",
                "",
                0,
                StatusTier.STATUS_TIERS.get(4),
                new ArrayList<>()
        );

        // Get a workflow stub using the same task queue the worker uses.
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setTaskQueue(testWorkflowRule.getTaskQueue())
                        .setWorkflowId(CustomerLoyaltyWorkflow.workflowIdForCustomer(customer))
                        .setWorkflowIdReusePolicy(
                                WorkflowIdReusePolicy.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY)
                        .build();
        CustomerLoyaltyWorkflow workflow =
                testWorkflowRule
                        .getWorkflowClient()
                        .newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        WorkflowClient.start(workflow::customerLoyalty, customer);

        int order = 0;
        var guest = new Customer("guest");
        testWorkflowRule.getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(order++),
                        () -> workflow.inviteGuest(guest));

        testWorkflowRule.getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(order++),
                        () -> workflow.inviteGuest(guest));

        testWorkflowRule.getTestEnvironment()
                .registerDelayedCallback(Duration.ofSeconds(order++),
                        workflow::cancelAccount);

        testWorkflowRule.getTestEnvironment().sleep(Duration.ofSeconds(order + 1));

        verify(activities, times(1))
                .sendEmail(EmailStrings.EMAIL_GUEST_INVITED);
        verify(activities, times(1))
                .sendEmail(EmailStrings.EMAIL_GUEST_CANCELED);

        WorkflowStub.fromTyped(workflow).getResult(String.class);
        testWorkflowRule.getTestEnvironment().shutdown();
    }
}
