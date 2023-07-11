package actorworkflow;

import io.temporal.activity.ActivityInterface;

@ActivityInterface
public interface CustomerLoyaltyActivities {
    void sendEmail(String body);
}