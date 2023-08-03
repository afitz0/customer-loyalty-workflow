package actorworkflow;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonInclude.Include;

import java.util.ArrayList;
import java.util.Objects;

@JsonInclude(Include.NON_NULL)
@JsonIgnoreProperties(ignoreUnknown = true)
public final class Customer {
    private String customerId;
    private String name;
    private int loyaltyPoints;
    private StatusTier status;
    private ArrayList<Customer> guests;

    public Customer() {
    }

    public Customer(String customerId) {
        this(customerId,
                "",
                0,
                StatusTier.STATUS_TIERS.get(0),
                new ArrayList<>());
    }

    public Customer(String customerId, String name, int loyaltyPoints, StatusTier status, ArrayList<Customer> guests) {
        if (status == null) {
            status = StatusTier.STATUS_TIERS.get(0);
        }
        if (guests == null) {
            guests = new ArrayList<>();
        }
        this.customerId = customerId;
        this.name = name;
        this.loyaltyPoints = loyaltyPoints;
        this.status = status;
        this.guests = guests;
    }

    public Customer withPoints(int points) {
        this.loyaltyPoints = points;
        return this;
    }

    public Customer withStatus(StatusTier status) {
        this.status = status;
        return this;
    }


    public String customerId() {
        return customerId;
    }

    public String name() {
        return name;
    }

    public int loyaltyPoints() {
        return loyaltyPoints;
    }

    public StatusTier status() {
        return status;
    }

    public ArrayList<Customer> guests() {
        return guests;
    }

    @Override
    public boolean equals(Object obj) {
        if (obj == this) return true;
        if (obj == null || obj.getClass() != this.getClass()) return false;
        var that = (Customer) obj;
        return Objects.equals(this.customerId, that.customerId) &&
                Objects.equals(this.name, that.name) &&
                this.loyaltyPoints == that.loyaltyPoints &&
                Objects.equals(this.status, that.status) &&
                Objects.equals(this.guests, that.guests);
    }

    @Override
    public int hashCode() {
        return Objects.hash(customerId, name, loyaltyPoints, status, guests);
    }

    @Override
    public String toString() {
        return "Customer[" +
                "customerId=" + customerId + ", " +
                "name=" + name + ", " +
                "loyaltyPoints=" + loyaltyPoints + ", " +
                "status=" + status + ", " +
                "guests=" + guests + ']';
    }

    public static boolean canAddGuest(Customer customer) {
        return customer.guests().size() < customer.status().guestsAllowed();
    }
}
