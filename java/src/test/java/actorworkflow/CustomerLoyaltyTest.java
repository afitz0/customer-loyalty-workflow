package actorworkflow;

import io.temporal.api.enums.v1.WorkflowIdReusePolicy;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowExecutionAlreadyStarted;
import io.temporal.client.WorkflowOptions;
import io.temporal.client.WorkflowStub;
import io.temporal.testing.TestWorkflowRule;
import io.temporal.testing.WorkflowReplayer;
import org.junit.Rule;
import org.junit.Test;

import java.io.File;
import java.time.Duration;
import java.util.ArrayList;
import java.util.Objects;

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
        CustomerLoyaltyActivities activities = mock(CustomerLoyaltyActivities.class);
        testWorkflowRule.getWorker().registerActivitiesImplementations(activities);
        testWorkflowRule.getTestEnvironment().start();

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

        workflow.cancelAccount();
        testWorkflowRule.getTestEnvironment().sleep(Duration.ofSeconds(1));

        WorkflowStub.fromTyped(workflow).getResult(String.class);
        testWorkflowRule.getTestEnvironment().shutdown();
    }

    @Test
    public void testAddGuest() {
        CustomerLoyaltyActivities activities = mock(CustomerLoyaltyActivities.class);
        testWorkflowRule.getWorker().registerActivitiesImplementations(activities);
        testWorkflowRule.getTestEnvironment().start();

        // Get a workflow stub using the same task queue the worker uses.
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setTaskQueue(testWorkflowRule.getTaskQueue())
                        .setWorkflowIdReusePolicy(
                                WorkflowIdReusePolicy.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE)
                        .setWorkflowId(Shared.WORKFLOW_ID_FORMAT.formatted("host"))
                        .build();
        CustomerLoyaltyWorkflow workflow =
                testWorkflowRule
                        .getWorkflowClient()
                        .newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        // Start workflow asynchronously to not use another thread to signal.
        var customer = new Customer("host", "", 0, StatusTier.STATUS_TIERS.get(4), new ArrayList<>());
        WorkflowClient.start(workflow::customerLoyalty, customer);

        var guest = new Customer("guest");
        workflow.inviteGuest(guest);

        CustomerLoyaltyWorkflow child = testWorkflowRule
                .getWorkflowClient()
                .newWorkflowStub(CustomerLoyaltyWorkflow.class,
                        WorkflowOptions.newBuilder()
                                .setTaskQueue(testWorkflowRule.getTaskQueue())
                                .setWorkflowId(Shared.WORKFLOW_ID_FORMAT.formatted(guest.customerId()))
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
        CustomerLoyaltyActivities activities = mock(CustomerLoyaltyActivities.class);
        testWorkflowRule.getWorker().registerActivitiesImplementations(activities);
        testWorkflowRule.getTestEnvironment().start();

        // Get a workflow stub using the same task queue the worker uses.
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setTaskQueue(testWorkflowRule.getTaskQueue())
                        .setWorkflowId(Shared.WORKFLOW_ID_FORMAT.formatted("host"))
                        .setWorkflowIdReusePolicy(
                                WorkflowIdReusePolicy.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE)
                        .build();
        CustomerLoyaltyWorkflow workflow =
                testWorkflowRule
                        .getWorkflowClient()
                        .newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        var customer = new Customer("host", "", 0, StatusTier.STATUS_TIERS.get(4), new ArrayList<>());
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

        CustomerLoyaltyWorkflow child = testWorkflowRule
                .getWorkflowClient()
                .newWorkflowStub(CustomerLoyaltyWorkflow.class,
                        WorkflowOptions.newBuilder()
                                .setTaskQueue(testWorkflowRule.getTaskQueue())
                                .setWorkflowId(Shared.WORKFLOW_ID_FORMAT.formatted(guest.customerId()))
                                .build());

        testWorkflowRule.getTestEnvironment().registerDelayedCallback(Duration.ofSeconds(order++), () -> {
            // "start" the workflow, to make sure we have the current execution, but expect it to throw
            try {
                WorkflowClient.start(child::customerLoyalty, guest);
            } catch (WorkflowExecutionAlreadyStarted ignored) {
            }
            child.cancelAccount();
        });

        testWorkflowRule.getTestEnvironment().sleep(Duration.ofSeconds(order + 1));

        verify(activities, times(1))
                .sendEmail(EmailStrings.EMAIL_GUEST_INVITED);
        verify(activities, times(1))
                .sendEmail(EmailStrings.EMAIL_GUEST_MIN_STATUS.formatted(StatusTier.STATUS_TIERS.get(3).name()));

        WorkflowStub.fromTyped(workflow).getResult(String.class);
        WorkflowStub.fromTyped(child).getResult(String.class);
        testWorkflowRule.getTestEnvironment().shutdown();
    }

    @Test(expected = Test.None.class)
    public void testSimpleReplay() throws Exception {
        ClassLoader classLoader = getClass().getClassLoader();
        File file = new File(Objects.requireNonNull(classLoader.getResource("simple_replay.json")).getFile());

        WorkflowReplayer.replayWorkflowExecution(file, CustomerLoyaltyWorkflowImpl.class);
    }
}
