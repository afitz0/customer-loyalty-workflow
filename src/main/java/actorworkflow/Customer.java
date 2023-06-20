package actorworkflow;

import java.util.ArrayList;

public class Customer {
    private String name;
    private String customerId;
    private int loyaltyPoints;
    private StatusTier status;
    private ArrayList<Customer> guests = new ArrayList<>();

    public Customer(String customerId) {
        this.customerId = customerId;
    }

    public Customer(String name, String customerId) {
        this.customerId = customerId;
        this.name = name;
    }

    public Customer(String name, String customerId, int loyaltyPoints) {
        this.name = name;
        this.customerId = customerId;
        this.loyaltyPoints = loyaltyPoints;
    }

    public Customer(String name, String customerId, int loyaltyPoints, StatusTier status) {
        this.name = name;
        this.customerId = customerId;
        this.loyaltyPoints = loyaltyPoints;
        this.status = status;
    }

    public Customer(String name, String customerId, int loyaltyPoints, StatusTier status, ArrayList<Customer> guests) {
        this.name = name;
        this.customerId = customerId;
        this.loyaltyPoints = loyaltyPoints;
        this.status = status;
        this.guests = guests;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getCustomerId() {
        return customerId;
    }

    public void setCustomerId(String customerId) {
        this.customerId = customerId;
    }

    public int getLoyaltyPoints() {
        return loyaltyPoints;
    }

    public void setLoyaltyPoints(int loyaltyPoints) {
        this.loyaltyPoints = loyaltyPoints;
    }

    public StatusTier getStatus() {
        return status;
    }

    public void setStatus(StatusTier status) {
        this.status = status;
    }

    public ArrayList<Customer> getGuests() {
        return guests;
    }

    public void setGuests(ArrayList<Customer> guests) {
        this.guests = guests;
    }

    public void addGuest(Customer customer) {
        this.guests.add(customer);
    }

    public boolean canAddGuest() {
        return guests.size() < status.guestsAllowed();
    }
}
