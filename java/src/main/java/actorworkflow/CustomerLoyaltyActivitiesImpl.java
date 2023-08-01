package actorworkflow;

import io.temporal.api.enums.v1.WorkflowIdReusePolicy;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowExecutionAlreadyStarted;
import io.temporal.client.WorkflowOptions;
import io.temporal.client.WorkflowStub;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import static actorworkflow.CustomerLoyaltyWorkflow.workflowIdForCustomer;

public class CustomerLoyaltyActivitiesImpl implements CustomerLoyaltyActivities {
    private final WorkflowClient client;

    public CustomerLoyaltyActivitiesImpl(WorkflowClient client) {
        this.client = client;
    }

    private static final Logger logger = LoggerFactory.getLogger(CustomerLoyaltyActivitiesImpl.class);

    @Override
    public void sendEmail(String body) {
        logger.info("Sending email: '{}'.", body);
    }

    @Override
    public boolean startGuestWorkflow(Customer guest, String taskQueue) {
        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setWorkflowId(workflowIdForCustomer(guest))
                        .setTaskQueue(taskQueue)
                        .setWorkflowIdReusePolicy(WorkflowIdReusePolicy.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY)
                        .build();
        WorkflowStub untypedWorkflowStub = client.newUntypedWorkflowStub(
                "CustomerLoyaltyWorkflow",
                workflowOptions
        );

        try {
            untypedWorkflowStub.signalWithStart(
                    "ensureMinimumStatus",
                    new Object[]{guest.status()},
                    new Object[]{guest}
            );
        } catch (WorkflowExecutionAlreadyStarted e) {
            return false;
        }

        return true;
    }

}
