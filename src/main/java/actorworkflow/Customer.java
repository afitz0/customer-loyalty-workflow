package actorworkflow;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonInclude.Include;

import java.util.ArrayList;

@JsonInclude(Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
public record Customer(
        String customerId,
        String name,
        int loyaltyPoints,
        StatusTier status,
        ArrayList<Customer> guests
) {
    public Customer(String customerId) {
        this(customerId,
                "",
                0,
                StatusTier.STATUS_TIERS.get(0),
                new ArrayList<>());
    }

    public Customer {
        if (status == null) {
            status = StatusTier.STATUS_TIERS.get(0);
        }
        if (guests == null) {
            guests = new ArrayList<>();
        }
    }

    public Customer withPoints(int points) {
        return new Customer(
                customerId,
                name,
                points,
                status,
                guests
        );
    }

    public Customer withStatus(StatusTier status) {
        return new Customer(
                customerId,
                name,
                loyaltyPoints,
                status,
                guests
        );
    }

    public static boolean canAddGuest(Customer customer) {
        return customer.guests().size() < customer.status().guestsAllowed();
    }
}
