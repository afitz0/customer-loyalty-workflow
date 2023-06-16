package actorworkflow;

public class StatusTier {
    private String name;
    private int minimumPoints;
    private int guestsAllowed;

    public StatusTier(String name, int minimumPoints, int guestsAllowed) {
        this.name = name;
        this.minimumPoints = minimumPoints;
        this.guestsAllowed = guestsAllowed;
    }

    public String getName() {
        return name;
    }

    public int getMinimumPoints() {
        return minimumPoints;
    }

    public int getGuestsAllowed() {
        return guestsAllowed;
    }
}
