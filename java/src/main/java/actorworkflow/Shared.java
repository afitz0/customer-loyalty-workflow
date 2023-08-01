package actorworkflow;

public final class Shared {
    private Shared() {}

    static final String TASK_QUEUE_NAME = "CustomerLoyaltyTaskQueue";
    static final int HISTORY_THRESHOLD = 10_000;
}
