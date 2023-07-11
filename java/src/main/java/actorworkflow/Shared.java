package actorworkflow;

public interface Shared {
    String TASK_QUEUE_NAME = "CustomerLoyaltyTaskQueue";

    int HISTORY_THRESHOLD = 10_000;

    String WORKFLOW_ID_FORMAT = "customer-%s";

    StatusTier[] STATUS_TIERS = {
            new StatusTier("Member", 0, 0),
            new StatusTier("Bronze", 500, 1),
            new StatusTier("Silver", 1_000, 2),
            new StatusTier("Gold", 2_000, 5),
            new StatusTier("Platinum", 5_000, 10),
    };
}
