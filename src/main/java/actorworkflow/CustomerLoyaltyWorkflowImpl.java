package actorworkflow;

import io.temporal.activity.ActivityOptions;
import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.api.enums.v1.ParentClosePolicy;
import io.temporal.client.WorkflowExecutionAlreadyStarted;
import io.temporal.workflow.*;
import org.slf4j.Logger;

import java.time.Duration;
import java.util.ArrayList;

public class CustomerLoyaltyWorkflowImpl implements CustomerLoyaltyWorkflow {

    private static final Logger logger = Workflow.getLogger(CustomerLoyaltyWorkflowImpl.class);

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
        System.out.println("Customer passed in is: " + customer);
        this.customer = customer;
        WorkflowInfo info = Workflow.getInfo();

        logger.info("Started workflow. Customer is {}", this.customer);

        if (info.getContinuedExecutionRunId().isEmpty()) {
            String tier = this.customer.status().name();
            activities.sendEmail("Welcome to our loyalty program! You're starting out at the '%s' tier."
                    .formatted(tier));
        }

        // block on everything
        while (true) {
            Workflow.await(() -> !accountActive || info.getHistoryLength() > Shared.HISTORY_THRESHOLD);
            if (accountActive) {
                logger.info("Account still active, history limit crossed limit; continuing-as-new?");
                Workflow.continueAsNew(this.customer);
            } else {
                logger.info("Account canceled. Closing workflow.");
                return "Done";
            }
        }
    }

    @Override
    public void addLoyaltyPoints(int pointsToAdd) {
        customer = customer.withPoints(customer.loyaltyPoints() + pointsToAdd);
        logger.info("Added {} points to customer. Loyalty points now {}",
                pointsToAdd, customer.loyaltyPoints());

        StatusTier tierToPromoteTo = StatusTier.getMaxTier(customer.loyaltyPoints());

        if (customer.status().minimumPoints() < tierToPromoteTo.minimumPoints()) {
            logger.info("Promoting customer!");
            customer = customer.withStatus(tierToPromoteTo);
            activities.sendEmail("Congratulations! You've been promoted to the '%s' tier!"
                    .formatted(tierToPromoteTo.name()));
        }
    }

    @Override
    public void inviteGuest(Customer guest) {
        logger.info("Attempting to invite guest {}", guest);
        if (Customer.canAddGuest(customer)) {
            logger.info("Customer is allowed to invite guests; attempting to start guest workflow.");
            customer.guests().add(guest);

            StatusTier guestMinStatus = StatusTier.previous(customer.status());
            guest = guest.withStatus(guestMinStatus);

            String guestWorkflowId = Shared.WORKFLOW_ID_FORMAT.formatted(guest.customerId());
            ChildWorkflowOptions options =
                    ChildWorkflowOptions.newBuilder()
                            .setWorkflowId(guestWorkflowId)
                            .setParentClosePolicy(ParentClosePolicy.PARENT_CLOSE_POLICY_ABANDON)
                            .build();

            CustomerLoyaltyWorkflow child = Workflow.newChildWorkflowStub(CustomerLoyaltyWorkflow.class, options);

            boolean alreadyStarted = false;
            try {
                Promise<WorkflowExecution> childExecution = Workflow.getWorkflowExecution(child);
                Async.procedure(child::customerLoyalty, guest);

                // Wait for child to start
                WorkflowExecution ce = childExecution.get();
            } catch (WorkflowExecutionAlreadyStarted e) {
                logger.info("Guest customer workflow already started and is a direct child.");
                alreadyStarted = true;
            } catch (Exception e) {
                if (e.getCause() instanceof WorkflowExecutionAlreadyStarted) {
                    logger.info("Guest customer workflow already started.");
                } else {
                    throw e;
                }
                alreadyStarted = true;
            }

            if (alreadyStarted) {
                // Reset child to ensure we're actually working with the latest running execution
                child = Workflow.newExternalWorkflowStub(CustomerLoyaltyWorkflow.class, guestWorkflowId);

//            ExternalWorkflowStub childWorkflowToSignal = Workflow.newUntypedExternalWorkflowStub(guestWorkflowId);
                logger.info("Signaling to ensure that they're at least \"{}\" status", guestMinStatus.name());
//            childWorkflowToSignal.signal("ensureMinimumStatus", guestMinStatus);
                child.ensureMinimumStatus(guestMinStatus);

                activities.sendEmail("Your guest already has an account, but we've made sure they're at least '%s' status!"
                        .formatted(guestMinStatus.name()));
            } else {
                activities.sendEmail("Your guest has been invited!");
            }
        }
    }

    @Override
    public void ensureMinimumStatus(StatusTier status) {
        WorkflowInfo info = Workflow.getInfo();
        logger.info("Ensuring that status is at minimum {}.", status.name());
        while (customer.status().minimumPoints() < status.minimumPoints()) {
            customer = customer.withStatus(StatusTier.next(customer.status()));
        }

        customer = customer.withPoints(Math.max(customer.loyaltyPoints(), status.minimumPoints()));
    }

    @Override
    public void cancelAccount() {
        this.accountActive = false;
    }

    @Override
    public StatusTier getStatus() {
        return customer.status();
    }

    @Override
    public ArrayList<Customer> getGuests() {
        return customer.guests();
    }
}
