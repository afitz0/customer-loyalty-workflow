package actorworkflow;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Objects;

public final class StatusTier {
    public static final List<StatusTier> STATUS_TIERS = new ArrayList<>(Arrays.asList(
            new StatusTier("Member", 0, 0, 0),
            new StatusTier("Bronze", 500, 1, 1),
            new StatusTier("Silver", 1_000, 2, 2),
            new StatusTier("Gold", 2_000, 5, 3),
            new StatusTier("Platinum", 5_000, 10, 4)
    ));

    private final String name;
    private final int minimumPoints;
    private final int guestsAllowed;
    private final int level;

    public StatusTier() {
        this(STATUS_TIERS.get(0));
    }

    public StatusTier(StatusTier tier) {
        this.name = tier.name();
        this.minimumPoints = tier.minimumPoints();
        this.guestsAllowed = tier.guestsAllowed();
        this.level = tier.level();
    }

    public StatusTier(String name, int minimumPoints, int guestsAllowed, int level) {
        this.name = name;
        this.minimumPoints = minimumPoints;
        this.guestsAllowed = guestsAllowed;
        this.level = level;
    }

    public static StatusTier next(StatusTier tier) {
        int nextIndex = STATUS_TIERS.indexOf(tier) + 1;
        return STATUS_TIERS.get(Math.min(STATUS_TIERS.size(), nextIndex));
    }

    public static StatusTier previous(StatusTier tier) {
        int prevIndex = STATUS_TIERS.indexOf(tier) - 1;
        return STATUS_TIERS.get(Math.max(0, prevIndex));
    }

    public static StatusTier getMaxTier(int points) {
        for (int i = STATUS_TIERS.size() - 1; i >= 0; i--) {
            StatusTier tier = STATUS_TIERS.get(i);
            if (points >= tier.minimumPoints()) {
                return tier;
            }
        }
        return STATUS_TIERS.get(0);
    }

    public String name() {
        return name;
    }

    public int minimumPoints() {
        return minimumPoints;
    }

    public int guestsAllowed() {
        return guestsAllowed;
    }

    public int level() {
        return level;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        StatusTier that = (StatusTier) o;
        return minimumPoints == that.minimumPoints && guestsAllowed == that.guestsAllowed && level == that.level && Objects.equals(name, that.name);
    }

    @Override
    public int hashCode() {
        return Objects.hash(name, minimumPoints, guestsAllowed, level);
    }

    @Override
    public String toString() {
        return "StatusTier{" +
                "name='" + name + '\'' +
                ", minimumPoints=" + minimumPoints +
                ", guestsAllowed=" + guestsAllowed +
                ", level=" + level +
                '}';
    }
}
