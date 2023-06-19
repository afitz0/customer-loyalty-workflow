package actorworkflow;

public interface CustomerActor {
    void addPoints(int pointsToAdd);

    void inviteGuest(Customer guest);

    void cancelAccount();

    StatusTier getStatus();
}