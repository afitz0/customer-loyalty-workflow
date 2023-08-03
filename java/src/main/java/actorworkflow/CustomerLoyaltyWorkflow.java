package actorworkflow;

import io.temporal.workflow.QueryMethod;
import io.temporal.workflow.SignalMethod;
import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;

import java.util.ArrayList;

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
    void ensureMinimumStatus(StatusTier status);

    @SignalMethod
    void cancelAccount();

    @QueryMethod
    StatusTier getStatus();

    @QueryMethod
    ArrayList<Customer> getGuests();

    @QueryMethod
    Customer getCustomer();

    static String workflowIdForCustomer(Customer customer) {
        return "customer-%s".formatted(customer.customerId());
    }
}