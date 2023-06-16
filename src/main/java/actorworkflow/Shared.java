package actorworkflow;

public interface Shared {
    static final public String TASK_QUEUE_NAME = "BugfixDemoTaskQueue";

    static final public int HISTORY_THRESHOLD = 10_000;

    static final public String WORKFLOW_ID_FORMAT = "customer-%s";

    static final public StatusTier[] STATUS_TIERS = {
            new StatusTier("Member", 0, 0),
            new StatusTier("Bronze", 500, 1),
            new StatusTier("Silver", 1_000, 2),
            new StatusTier("Gold", 2_000, 5),
            new StatusTier("Platinum", 5_000, 10),
    };
}
