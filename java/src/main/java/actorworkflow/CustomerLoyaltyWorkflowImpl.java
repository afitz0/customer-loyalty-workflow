package actorworkflow;

import io.temporal.activity.ActivityOptions;
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
        this.customer = customer;
        WorkflowInfo info = Workflow.getInfo();

        logger.info("Started workflow. Customer: {}.", this.customer);

        if (info.getContinuedExecutionRunId().isEmpty()) {
            String tier = this.customer.status().name();
            activities.sendEmail(EmailStrings.EMAIL_WELCOME.formatted(tier));
        }

        // block on everything
        while (true) {
            Workflow.await(() -> !accountActive || info.getHistoryLength() > Shared.HISTORY_THRESHOLD);
            if (accountActive) {
                logger.info("Account still active, history size crossed limit; continuing-as-new.");
                Workflow.continueAsNew(this.customer);
            } else {
                logger.info("Account canceled. Closing workflow.");
                return "Done";
            }
        }
    }

    @Override
    public void addLoyaltyPoints(int pointsToAdd) {
        int previousLevel = customer.status().level();

        customer = activities.addPoints(customer, pointsToAdd);
        logger.info("Added {} points to customer. Total loyalty points now {}.",
                pointsToAdd, customer.loyaltyPoints());

        if (previousLevel < customer.status().level()) {
            activities.sendEmail(EmailStrings.EMAIL_PROMOTED.formatted(customer.status().name()));
        }
    }

    @Override
    public void inviteGuest(Customer guest) {
        logger.info("Checking to see if customer can invite guests.");
        if (Customer.canAddGuest(customer)) {
            logger.info("Customer is allowed to invite guests; attempting to start workflow for guest ID {}.",
                    guest.customerId());
            customer.guests().add(guest);

            StatusTier guestMinStatus = StatusTier.previous(customer.status());
            guest.withStatus(guestMinStatus);

            boolean started = activities.startGuestWorkflow(guest, Workflow.getInfo().getTaskQueue());
            if (started) {
                activities.sendEmail(EmailStrings.EMAIL_GUEST_INVITED);
            } else {
                activities.sendEmail(EmailStrings.EMAIL_GUEST_CANCELED);
            }
        }
    }

    @Override
    public void ensureMinimumStatus(StatusTier status) {
        logger.info("Ensuring that status is at minimum {}.", status.name());
        while (customer.status().minimumPoints() < status.minimumPoints()) {
            customer = customer.withStatus(StatusTier.next(customer.status()));
        }

        customer = customer.withPoints(Math.max(customer.loyaltyPoints(), status.minimumPoints()));
    }

    @Override
    public void cancelAccount() {
        this.accountActive = false;
        activities.sendEmail(EmailStrings.EMAIL_CANCEL_ACCOUNT);
    }

    @Override
    public StatusTier getStatus() {
        return customer.status();
    }

    @Override
    public ArrayList<Customer> getGuests() {
        return customer.guests();
    }

    @Override
    public Customer getCustomer() {
        return customer;
    }
}
