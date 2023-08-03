package actorworkflow;

import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.serviceclient.WorkflowServiceStubs;

public class Starter {
    public static void main(String[] args) {
        WorkflowServiceStubs service = WorkflowServiceStubs.newLocalServiceStubs();
        WorkflowClient client = WorkflowClient.newInstance(service);

        Customer customer = new Customer("123");
        customer.withStatus(StatusTier.STATUS_TIERS.get(3));

        WorkflowOptions workflowOptions =
                WorkflowOptions.newBuilder()
                        .setWorkflowId(CustomerLoyaltyWorkflow.workflowIdForCustomer(customer))
                        .setTaskQueue(Shared.TASK_QUEUE_NAME)
                        .build();
        CustomerLoyaltyWorkflow workflow = client.newWorkflowStub(CustomerLoyaltyWorkflow.class, workflowOptions);

        WorkflowExecution we =  WorkflowClient.start(workflow::customerLoyalty, customer);

        System.out.println("Started the workflow. See the Worker's console for output.");
        System.out.printf("Workflow ID: %s\nRun ID: %s\n", we.getWorkflowId(), we.getRunId());

        System.exit(0);
    }
}