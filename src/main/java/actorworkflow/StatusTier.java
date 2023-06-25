package actorworkflow;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

public record StatusTier(String name, int minimumPoints, int guestsAllowed) {
    public static final List<StatusTier> STATUS_TIERS = new ArrayList<>(Arrays.asList(
            new StatusTier("Member", 0, 0),
            new StatusTier("Bronze", 500, 1),
            new StatusTier("Silver", 1_000, 2),
            new StatusTier("Gold", 2_000, 5),
            new StatusTier("Platinum", 5_000, 10)
    ));

    public static StatusTier next(StatusTier tier) {
        int nextIndex = STATUS_TIERS.indexOf(tier) + 1;
        return STATUS_TIERS.get(Math.min(STATUS_TIERS.size(), nextIndex));
    }

    public static StatusTier previous(StatusTier tier) {
        int prevIndex = STATUS_TIERS.indexOf(tier) - 1;
        return STATUS_TIERS.get(Math.max(0, prevIndex));
    }

    public static StatusTier getMaxTier(int points) {
        for (int i = STATUS_TIERS.size() - 1; i >=0; i--) {
            StatusTier tier = STATUS_TIERS.get(i);
            if (points >= tier.minimumPoints()) {
                return tier;
            }
        }
        return STATUS_TIERS.get(0);
    }
}
