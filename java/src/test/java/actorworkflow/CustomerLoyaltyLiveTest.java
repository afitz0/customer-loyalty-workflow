package actorworkflow;

import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.serviceclient.WorkflowServiceStubs;
import io.temporal.worker.WorkerFactory;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

/**
 * Test that requires a local Temporal Server to be running. Starts a unique customer workflow, invites a guest and
 * then verifies that it was done successfully.
 */
public class CustomerLoyaltyLiveTest {
    private static final String TASK_QUEUE = "testing-customer-loyalty";

    public static final WorkflowServiceStubs service = WorkflowServiceStubs.newLocalServiceStubs();
    public static final WorkflowClient client = WorkflowClient.newInstance(service);
    public static final WorkerFactory factory = WorkerFactory.newInstance(client);

    private static final Logger logger = LoggerFactory.getLogger(Worker.class);

    public static void main(String[] args) {
        // Start a worker on a dedicated testing task queue
        io.temporal.worker.Worker worker = factory.newWorker(TASK_QUEUE);
        worker.registerWorkflowImplementationTypes(CustomerLoyaltyWorkflowImpl.class);
        worker.registerActivitiesImplementations(new CustomerLoyaltyActivitiesImpl(client));

        logger.debug("Starting worker.");
        factory.start();

        boolean success = true;

        try {
            runTest();
        } catch (InterruptedException ignored) {
        } catch (AssertionError e) {
            System.out.println("Tests failed: " + e.getMessage());
            success = false;
        } finally {
            factory.shutdown();
        }

        if (!success) {
            System.exit(1);
        }
    }

    private static void runTest() throws InterruptedException, AssertionError {
        Customer customer = new Customer("host-" + UUID.randomUUID());
        customer.withStatus(StatusTier.STATUS_TIERS.get(3));

        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setWorkflowId(CustomerLoyaltyWorkflow.workflowIdForCustomer(customer))
                        .setTaskQueue(TASK_QUEUE)
                        .build();
        CustomerLoyaltyWorkflow workflow = client.newWorkflowStub(
                CustomerLoyaltyWorkflow.class,
                workflowOptions
        );

        // start "host"/origin workflow
        WorkflowExecution we = WorkflowClient.start(workflow::customerLoyalty, customer);

        // Wait a moment for things to start
        Thread.sleep(1000);

        // signal invite guest
        String guestId = "guest-" + UUID.randomUUID();
        Customer guest = new Customer(guestId);
        workflow.inviteGuest(guest);

        logger.info("Started guest workflow, waiting for things to settle.");

        // Wait a moment for guest things to start
        Thread.sleep(1000);

        CustomerLoyaltyWorkflow guestWorkflow = client.newWorkflowStub(
                CustomerLoyaltyWorkflow.class,
                CustomerLoyaltyWorkflow.workflowIdForCustomer(guest)
        );

        logger.info("Querying for guest's status.");
        StatusTier guestStatus = guestWorkflow.getStatus();
        StatusTier expectedStatus = StatusTier.previous(customer.status());

        if (!guestStatus.equals(expectedStatus)) {
            throw new AssertionError("Guest status ('%s') does not match expected ('%s').".formatted(guestStatus, expectedStatus));
        }

        logger.info("Querying for host customer's guest list.");
        ArrayList<Customer> guestList = workflow.getGuests();

        List<String> idList = guestList.stream().map(Customer::customerId).toList();
        if (!idList.contains(guest.customerId())) {
            throw new AssertionError("Expected guest to be in customer's list (%s), but not found.".formatted(guestList));
        }

        Thread.sleep(1000);

        logger.info("Canceling both accounts");
        workflow.cancelAccount();
        guestWorkflow.cancelAccount();

        logger.info("Blocking for workflows to finish");
        workflow.customerLoyalty(customer);
        guestWorkflow.customerLoyalty(guest);
    }
}
