package actorworkflow;

import io.temporal.activity.ActivityInterface;

@ActivityInterface
public interface CustomerLoyaltyActivities {
    void sendEmail(String body);

    boolean startGuestWorkflow(Customer guest, String taskQueue);

    Customer addPoints(Customer customer, int pointsToAdd);
}