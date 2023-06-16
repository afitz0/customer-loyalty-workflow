package actorworkflow;

import io.temporal.activity.ActivityOptions;
import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.api.enums.v1.ParentClosePolicy;
import io.temporal.client.WorkflowExecutionAlreadyStarted;
import io.temporal.common.RetryOptions;
import io.temporal.workflow.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.Arrays;

public class CustomerLoyaltyWorkflowImpl implements CustomerLoyaltyWorkflow {

    private static final Logger logger = LoggerFactory.getLogger(CustomerLoyaltyWorkflowImpl.class);

    private final CustomerLoyaltyActivities activities =
            Workflow.newActivityStub(
                    CustomerLoyaltyActivities.class,
                    ActivityOptions.newBuilder()
                            .setStartToCloseTimeout(Duration.ofSeconds(2))
                            .build());

    boolean accountActive = true;

    Customer customer;

    @Override
    public String customerLoyalty(Customer customer) {
        this.customer = customer;
        WorkflowInfo info = Workflow.getInfo();

        if (info.getContinuedExecutionRunId().isEmpty()) {
            StatusTier tier = Shared.STATUS_TIERS[customer.getStatusLevel()];
            activities.sendEmail("Welcome to our loyalty program! You're starting out at the '%s' tier."
                    .formatted(tier.getName()));
        }

        // block on everything
        while (true) {
            Workflow.await(() -> !accountActive
                    || info.getHistoryLength() > Shared.HISTORY_THRESHOLD);
            if (accountActive) {
                logger.info("Account still active, history limit crossed limit; continuing-as-new.");
                Workflow.continueAsNew();
            } else {
                logger.info("Account canceled. Closing workflow.");
                return "Done";
            }
        }
    }

    @Override
    public void addLoyaltyPoints(int pointsToAdd) {
        customer.setLoyaltyPoints(customer.getLoyaltyPoints() + pointsToAdd);
        logger.info("Added {} points to customer. Loyalty points now {}",
                pointsToAdd, customer.getLoyaltyPoints());

        int nextLevel = customer.getStatusLevel() + 1;
        if (nextLevel < Shared.STATUS_TIERS.length) {
            StatusTier nextTier = Shared.STATUS_TIERS[nextLevel];
            if (customer.getLoyaltyPoints() >= nextTier.getMinimumPoints()) {
                logger.info("Promoting customer to next level!");
                customer.setStatusLevel(nextLevel);
                activities.sendEmail("Congratulations! You've been promoted to the '%s' tier!"
                        .formatted(nextTier.getName()));
            }
        }
    }

    @Override
    public void inviteGuest(Customer guest) {
        if (customer.canAddGuest()) {
            logger.info("Attempting to invite a guest.");
            customer.addGuest(guest);

            String guestWorkflowId = Shared.WORKFLOW_ID_FORMAT.formatted(guest.getCustomerId());
            ChildWorkflowOptions options =
                    ChildWorkflowOptions.newBuilder()
                            .setWorkflowId(guestWorkflowId)
                            .setParentClosePolicy(ParentClosePolicy.PARENT_CLOSE_POLICY_ABANDON)
                            .build();

            CustomerLoyaltyWorkflow child = Workflow.newChildWorkflowStub(CustomerLoyaltyWorkflow.class, options);

            try {
                Promise<WorkflowExecution> childExecution = Workflow.getWorkflowExecution(child);
                Async.procedure(child::customerLoyalty, guest);
                // Wait for child to start
                childExecution.get();
            } catch (WorkflowExecutionAlreadyStarted e) {
                logger.info("Guest customer workflow already started and is a direct child.");
            } catch (Exception e) {
                if (e.getCause() instanceof WorkflowExecutionAlreadyStarted) {
                    logger.info("Guest customer workflow already started.");
                } else {
                    throw e;
                }
            }

            ExternalWorkflowStub callRespondWorkflow = Workflow.newUntypedExternalWorkflowStub(guestWorkflowId);
            int guestMinLevel = Math.max(customer.getStatusLevel() - 1, 0);
            callRespondWorkflow.signal("ensureMinimumStatus", guestMinLevel);
        }
    }

    @Override
    public void ensureMinimumStatus(int statusLevel) {
        logger.info("Ensuring that status is at minimum {}.", statusLevel);
        if (customer.getStatusLevel() < statusLevel) {
            StatusTier tier = Shared.STATUS_TIERS[statusLevel];
            customer.setLoyaltyPoints(tier.getMinimumPoints());
            customer.setStatusLevel(statusLevel);
        }
    }

    @Override
    public void cancelAccount() {
        this.accountActive = false;
    }

    @Override
    public String getStatus() {
        return Shared.STATUS_TIERS[customer.getStatusLevel()].getName();
    }
}
