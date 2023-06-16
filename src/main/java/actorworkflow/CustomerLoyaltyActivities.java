package actorworkflow;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;

@ActivityInterface
public interface CustomerLoyaltyActivities {
    @ActivityMethod()
    void sendEmail(String body);
}