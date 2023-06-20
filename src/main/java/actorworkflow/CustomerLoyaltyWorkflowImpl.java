package actorworkflow;

import io.temporal.activity.ActivityOptions;
import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.api.enums.v1.ParentClosePolicy;
import io.temporal.client.WorkflowExecutionAlreadyStarted;
import io.temporal.workflow.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;

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
            String tier = customer.getStatus().name();
            activities.sendEmail("Welcome to our loyalty program! You're starting out at the '%s' tier."
                    .formatted(tier));
        }

        // block on everything
        while (true) {
            Workflow.await(() -> !accountActive);
            if (accountActive) {
                logger.info("Account still active, history limit crossed limit; continuing-as-new?");
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

        StatusTier tierToPromoteTo = StatusTier.getMaxTier(customer.getLoyaltyPoints());

        if (customer.getStatus().minimumPoints() < tierToPromoteTo.minimumPoints()) {
            logger.info("Promoting customer!");
            customer.setStatus(tierToPromoteTo);
            activities.sendEmail("Congratulations! You've been promoted to the '%s' tier!"
                    .formatted(tierToPromoteTo.name()));
        }
    }

    @Override
    public void inviteGuest(Customer guest) {
        if (customer.canAddGuest()) {
            logger.info("Attempting to invite a guest.");
            customer.addGuest(guest);
        }
    }

    @Override
    public void ensureMinimumStatus(StatusTier status) {
        logger.info("Ensuring that status is at minimum {}.", status.name());
        while (customer.getStatus().minimumPoints() < status.minimumPoints()) {
            customer.setStatus(StatusTier.next(customer.getStatus()));
        }

        customer.setLoyaltyPoints(Math.max(customer.getLoyaltyPoints(), status.minimumPoints()));
    }

    @Override
    public void cancelAccount() {
        this.accountActive = false;
    }

    @Override
    public StatusTier getStatus() {
        return customer.getStatus();
    }
}
