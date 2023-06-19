package actorworkflow;

import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;

public class StatusTier {
    private String name;
    private int minimumPoints;
    private int guestsAllowed;

    public static final List<StatusTier> STATUS_TIERS = new LinkedList<StatusTier>(Arrays.asList(
            new StatusTier("Member", 0, 0),
            new StatusTier("Bronze", 500, 1),
            new StatusTier("Silver", 1_000, 2),
            new StatusTier("Gold", 2_000, 5),
            new StatusTier("Platinum", 5_000, 10)
    ));

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
