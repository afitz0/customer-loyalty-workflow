package actorworkflow;

import java.util.ArrayList;

public class Customer {
    private String name;
    private String customerId;
    private int loyaltyPoints;
    private int statusLevel;
    private ArrayList<Customer> guests = new ArrayList<>();

    public Customer(String name, String customerId, int loyaltyPoints, int statusLevel) {
        this.name = name;
        this.customerId = customerId;
        this.loyaltyPoints = loyaltyPoints;
        this.statusLevel = statusLevel;
    }

    public Customer(String name, String customerId) {
        this.name = name;
        this.customerId = customerId;
    }

    public Customer(String name, String customerId, int loyaltyPoints, int statusLevel, ArrayList<Customer> guests) {
        this.name = name;
        this.customerId = customerId;
        this.loyaltyPoints = loyaltyPoints;
        this.statusLevel = statusLevel;
        this.guests = guests;
    }

    public Customer() {
        this.guests = new ArrayList<>();
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

    public int getStatusLevel() {
        return statusLevel;
    }

    public void setStatusLevel(int statusLevel) {
        this.statusLevel = statusLevel;
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
        StatusTier tier = Shared.STATUS_TIERS[statusLevel];
        return this.guests.size() < tier.getGuestsAllowed();
    }
}
