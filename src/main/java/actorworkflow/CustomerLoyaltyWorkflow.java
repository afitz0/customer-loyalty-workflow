package actorworkflow;

import io.temporal.workflow.QueryMethod;
import io.temporal.workflow.SignalMethod;
import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;

// Workflow interface
@WorkflowInterface
public interface CustomerLoyaltyWorkflow {
    @WorkflowMethod
    String customerLoyalty(Customer customer);

    @SignalMethod
    void addLoyaltyPoints(int pointsToAdd);

    @SignalMethod
    void inviteGuest(Customer guest);

    @SignalMethod
    void ensureMinimumStatus(int statusLevel);

    @SignalMethod
    void cancelAccount();

    @QueryMethod
    String getStatus();
}